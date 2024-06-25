# JWK Aggregator Plugin for KrakenD

This repository contains a Go plugin for KrakenD that serves JSON Web Keys (JWK) from multiple origins on a dedicated port.

The JWK aggregator plugin lets KrakenD validate tokens from multiple Identity Providers or realms within the same Identity Server.

Normally, KrakenD community edition can validate JWT tokens from only one Identity Provider per endpoint. However, in multi-tenant setups or during migrations, tokens might come from different providers or realms. The JWK aggregator plugin solves this problem by allowing KrakenD to handle tokens from various sources.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building the Plugin](#building-the-plugin)
3. [Configuring KrakenD](#configuring-krakend)

## Prerequisites

- KrakenD setup

## Building the Plugin

1. **Clone the Repository**

   ```sh
   git clone https://github.com/aphronio/krakend-jwk-aggregator-plugin.git
   cd jwk-aggregator-plugin
   ```

2. **Initialize a Go Module**

   Ensure you are in the `jwk-aggregator-plugin` directory and run:

   ```sh
   docker run -it -v "$PWD:/app" -w /app krakend/builder:<your-krakend-version> go mod init jwk_aggregator_plugin
   docker run -it -v "$PWD:/app" -w /app krakend/builder:<your-krakend-version> go mod tidy
   ```

3. **Compile the Plugin**

   Use Docker to compile the plugin into a shared object file (`.so`):

   ```sh
   docker run -it -v "$PWD:/app" -w /app krakend/builder:<your-krakend-version> go build -buildmode=plugin -o jwk_aggregator.so .
   ```

    >Notice: Replace `<your-krakend-version>` with the version of KrakenD you are using. For example, if you are using KrakenD version 2.5, replace <your-krakend-version> with 2.5.

4. **Copy the Plugin**

   Copy the compiled plugin to the KrakenD plugins folder:

   ```sh
   cp jwk_aggregator.so /opt/krakend/plugins/jwk_aggregator.so
   ```
## Configuring KrakenD

1. **Example Configuration File**

    Here is an example configuration file for injecting and configuring the krakend plugin we just built. You can configure the follwing fields:
    1. `port`: The port on which the plugin will serve the JWK keys
    2. `cache`: A boolean value that determines whether the plugin should cache the JWK keys
    3. `origins`: A list of URLs from which the plugin should fetch the JWK keys

    ```json
    {
        "version": 3,
        "$schema": "https://www.krakend.io/schema/krakend.json",
        "plugin": {
            "pattern": ".so",
            "folder": "/opt/krakend/plugins/"
        },
        "extra_config": {
            "plugin/http-server": {
                "name": ["jwk-aggregator"],
                "jwk-aggregator": {
                    "port": 9876,
                    "cache": true,
                    "origins": [
                        "https://keycloak/auth/realms/realm-1/protocol/openid-connect/certs",
                        "https://keycloak/auth/realms/realm-2/protocol/openid-connect/certs",
                        "https://provider1.tld/jwk.json",
                    ]
                }
            },
            "auth/validator": {
                "@comment": "Enable a JWK shared cache amongst all endpoints of 15 minutes",
                "shared_cache_duration": 900
            }
        },
        "endpoints": [
            {
                "endpoint": "/example",
                "extra_config": {
                    "auth/validator": {
                        "alg": "RS256",
                        "jwk_url": "http://localhost:9876",
                        "disable_jwk_security": true,
                        "cache": true
                    }
                },
                "backend": [{
                    "url_pattern": "/example"
                }]
            }
        ]
    }
    ```



## Conclusion

By following these steps, you have successfully built, configured, and run a Go plugin for KrakenD that serves JWK keys from multiple origins on a dedicated port. This setup ensures that JWK keys are served efficiently and securely, allowing your API gateway to validate JWTs with ease.
