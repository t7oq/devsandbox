# devsandbox

## Coding practices

After completing the task always run:

- `task test` - to run tests
- `task lint` - to run lint

## Tools to use

Tools management: mise
Create .mise.toml with all tools to be used and pin the dependencies.

Use `task` for setting up automation: create a taskfile which will cover build, lint, test, running the appliication.

CI: repository will be hosted at gitea. Update it to correlate with current project when needed.

Linters: golangci + go default linters.

## TODOs

- refactor tools to more modular approach
  - each tool should build its own paths/envs to be used
- add telemetry sending:
  - via otel - send logs/metrics/traces to otlp endpoint. metrics and traces TBD, logs are must
  - syslog - host and remote options
- capture more audit data (e.g. files access, attempts to access restricted files, attempts to access network etc)
- http filtering
  - ability to define whitelist/blacklist/ask mode for http(s) requests
- configuration support
  - support config file in toml
  - allow to configure tools to be enabled, proxy config
    - allow to generate config from logs for proxy
    - allow to generate config for tools based on detected tools availability
  - allow to configure per-project settings
