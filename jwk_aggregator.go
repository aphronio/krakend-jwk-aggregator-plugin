// SPDX-License-Identifier: Apache-2.0

package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "sync"
    "time"
)

// pluginName is the plugin name
var pluginName = "jwk-aggregator"

// HandlerRegisterer is the symbol the plugin loader will try to load. It must implement the Registerer interface
var HandlerRegisterer = registerer(pluginName)

type registerer string

func (r registerer) RegisterHandlers(f func(
    name string,
    handler func(context.Context, map[string]interface{}, http.Handler) (http.Handler, error),
)) {
    f(string(r), r.registerHandlers)
}

func (r registerer) registerHandlers(_ context.Context, extra map[string]interface{}, h http.Handler) (http.Handler, error) {
    // Configuration from KrakenD config
    config, ok := extra[pluginName].(map[string]interface{})
    if !ok {
        return h, errors.New("configuration not found")
    }

    // Get the origins and cache configuration
    origins, _ := config["origins"].([]interface{})
    originStrings := make([]string, len(origins))
    for i, v := range origins {
        originStrings[i], _ = v.(string)
    }
    cacheEnabled, _ := config["cache"].(bool)

    // Initialize the cache and refresh if enabled
    if cacheEnabled {
        go cacheRefresher(originStrings, 15*time.Minute)
    }

    // Return the actual handler wrapping or your custom logic so it can be used as a replacement for the default HTTP handler
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        if req.URL.Path != "/jwk-aggregator" {
            h.ServeHTTP(w, req)
            return
        }

        keys, err := fetchKeys(originStrings)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(keys)
        logger.Debug("request:", req.URL.Path)
    }), nil
}

func main() {}

var (
    cache     JWKSet
    cacheLock sync.Mutex
    lastFetch time.Time
)

// JWKSet represents a set of JWK keys.
type JWKSet struct {
    Keys []json.RawMessage `json:"keys"`
}

// fetchKeys fetches and aggregates JWK keys from multiple origins.
func fetchKeys(origins []string) (*JWKSet, error) {
    if cacheValid() {
        return &cache, nil
    }

    var keys JWKSet
    for _, origin := range origins {
        resp, err := http.Get(origin)
        if err != nil {
            logger.Error("Error fetching JWKs from", origin, ":", err)
            continue
        }

        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            logger.Error("Error reading response body from", origin, ":", err)
            continue
        }

        var keySet JWKSet
        if err := json.Unmarshal(body, &keySet); err != nil {
            logger.Error("Error unmarshalling JWKs from", origin, ":", err)
            continue
        }

        keys.Keys = append(keys.Keys, keySet.Keys...)
    }

    cacheLock.Lock()
    cache = keys
    lastFetch = time.Now()
    cacheLock.Unlock()

    return &keys, nil
}

// cacheValid checks if the cache is still valid.
func cacheValid() bool {
    cacheLock.Lock()
    defer cacheLock.Unlock()
    return time.Since(lastFetch) < 15*time.Minute
}

// cacheRefresher periodically refreshes the JWK cache.
func cacheRefresher(origins []string, interval time.Duration) {
    for {
        time.Sleep(interval)
        _, err := fetchKeys(origins)
        if err != nil {
            logger.Error("Error refreshing cache:", err)
        }
    }
}

// This logger is replaced by the RegisterLogger method to load the one from KrakenD
var logger Logger = noopLogger{}

func (registerer) RegisterLogger(v interface{}) {
    l, ok := v.(Logger)
    if (!ok) {
        return
    }
    logger = l
    logger.Debug(fmt.Sprintf("[PLUGIN: %s] Logger loaded", HandlerRegisterer))
}

type Logger interface {
    Debug(v ...interface{})
    Info(v ...interface{})
    Warning(v ...interface{})
    Error(v ...interface{})
    Critical(v ...interface{})
    Fatal(v ...interface{})
}

// Empty logger implementation
type noopLogger struct{}

func (n noopLogger) Debug(_ ...interface{})    {}
func (n noopLogger) Info(_ ...interface{})     {}
func (n noopLogger) Warning(_ ...interface{})  {}
func (n noopLogger) Error(_ ...interface{})    {}
func (n noopLogger) Critical(_ ...interface{}) {}
func (n noopLogger) Fatal(_ ...interface{})    {}
