# Go script to run a remote Jenkins job
Designed to be run in Docker and read parameters from the environment variables. If environment variables exist matching the prefix **PARAMETER_** the remote job is executed at the api **/buildWithParameters**; otherwise, the job is executed at **/build**.
## Parameters
- **JENKINS_USER** User account to access remote Jenkins
- **JENKINS_TOKEN** Token for account to access remote Jenkins
- **JENKINS_URI** Remote Jenkins address
- **JENKINS_JOB** Remote job to execute
- **PARAMETER_ENV_FOO** *(Optional)*

## Docker Run
```bash
docker run --rm \
  -e JENKINS_USER \
  -e JENKINS_TOKEN \
  -e JENKINS_JOB \
  -e JENKINS_URI \
  -e PARAMETER_ENV_FOO \
  jenkins-job
```

## Examples
Creates a simple Jenkins server and dummy job to execute.
