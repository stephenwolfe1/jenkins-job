package main

import (
  "os"
  "fmt"
  "time"
  "regexp"
  "strings"
  "net/url"
  "strconv"
  "net/http"
  "encoding/json"
  log "github.com/Sirupsen/logrus"
  )

func die(reason string) {
  log.Fatal(reason)
}

func getEnv(key, fallback string) string {
  if value, ok := os.LookupEnv(key); ok {
    return value
  }
  if fallback == "REQUIRED_ENV_VAR" {
    die(fmt.Sprintf("Environment variable %s required", key))
  }
  return fallback
}

func postJob(jenkins_user, jenkins_token, jenkins_uri, jenkins_job string, query_string url.Values) (string, string) {
  var url string

  client := &http.Client{}
  if len(query_string) > 0 {
    url = fmt.Sprintf("%s/job/%s/buildWithParameters", jenkins_uri, jenkins_job)
  } else {
    url = fmt.Sprintf("%s/job/%s/build", jenkins_uri, jenkins_job)
  }
  log.Debug("Posting job here: ", url)

  req, err := http.NewRequest("POST", url, strings.NewReader(query_string.Encode()))
  req.SetBasicAuth(jenkins_user, jenkins_token)
  resp, err := client.Do(req)

	if err != nil {
		die(err.Error())
	}
  defer resp.Body.Close()

  if resp.StatusCode != 201 {
    die(fmt.Sprintf("Request returned: %s", resp.Status))
  } else if resp.Header["Location"] == nil {
    die("Request accepted but no location returned")
  }

  loc := resp.Header["Location"][0]
  match, _ := regexp.MatchString("http.+queue.+", loc)
  if ! match {
    die("Request did not return a valid queue location")
  }

  log.Debug("Headers: ", loc)
  log.Debug("Response status: ", resp.StatusCode)
  pair := strings.Split(loc, "/")
  return loc, pair[5]
}

func wait_for_start(jenkins_user, jenkins_token, location, queue_id, interval, timeout string) string {

  client := &http.Client{}
  url := fmt.Sprintf("%s/api/json", location)
  log.Debug("Waiting for queued job to start")
  t, err := strconv.Atoi(timeout)
  i, err := strconv.Atoi(interval)
  if err != nil {
    die(err.Error())
  }
  limit := time.After(time.Duration(t) * time.Second)
	tick := time.Tick(time.Duration(i) * time.Second)

  for {
    select {
    case <-limit:
      die("Timeout elapsed while waiting for job to start")
    case <-tick:
      //time.Sleep(1 * time.Second)
      req, err := http.NewRequest("GET", url, nil)
      req.SetBasicAuth(jenkins_user, jenkins_token)
      resp, err := client.Do(req)
      if err != nil {
    		die(err.Error())
    	}
      defer resp.Body.Close()

      var result map[string]interface{}
      json.NewDecoder(resp.Body).Decode(&result)
      if result["executable"] == nil {
        log.Debug("Job has not started yet")
      } else {
        j_id := result["executable"].(map[string]interface{})["number"]
        if j_id == nil {
          die("Job started but job id could not be found")
        } else {
          return fmt.Sprintf("%v", int(j_id.(float64)))
        }
      }
    }
  }
}

func wait_for_complete(jenkins_user, jenkins_token, jenkins_uri, jenkins_job, job_id, interval, timeout string) string {

  client := &http.Client{}
  url := fmt.Sprintf("%s/job/%s/%s/api/json", jenkins_uri, jenkins_job, job_id)
  log.Debug("Waiting for running job to finish")
  t, err := strconv.Atoi(timeout)
  i, err := strconv.Atoi(interval)
  if err != nil {
    die(err.Error())
  }
  limit := time.After(time.Duration(t) * time.Second)
	tick := time.Tick(time.Duration(i) * time.Second)

  for {
    select {
    case <-limit:
      die("Timeout elapsed while waiting for job to finish")
    case <-tick:
      req, err := http.NewRequest("GET", url, nil)
      req.SetBasicAuth(jenkins_user, jenkins_token)
      resp, err := client.Do(req)
      if err != nil {
    		die(err.Error())
    	}
      defer resp.Body.Close()

      var result map[string]interface{}
      json.NewDecoder(resp.Body).Decode(&result)
      j_status := result["result"]
      switch j_status {
      case "SUCCESS":
        log.Info(fmt.Sprintf("Remote job logs can be viewed here: %s/job/%s/%s/consoleText", jenkins_uri, jenkins_job, job_id))
        return fmt.Sprintf("Job: %s Id: %s Completed Successfully", jenkins_job, job_id)
      case "FAILURE", "ABORTED":
        log.Info(fmt.Sprintf("Remote job logs can be viewed here: %s/job/%s/%s/consoleText", jenkins_uri, jenkins_job, job_id))
        die(fmt.Sprintf("Job: %s Id: %s %s", jenkins_job, job_id, j_status))
      default:
        log.Debug(fmt.Sprintf("Job: %s Id: %s Status: In Progress. Polling again in 1 secs", jenkins_job, job_id))
      }
    }
  }
}

func main() {
  log.SetFormatter(&log.TextFormatter{
    FullTimestamp: true,
    DisableLevelTruncation: true,
  })
  log.SetLevel(log.InfoLevel)

  //Testing
  //os.Setenv("JENKINS_USER", "user")
  //os.Setenv("JENKINS_URI", "http://localhost:80")
  //os.Setenv("JENKINS_JOB", "sample-deployer")
  //os.Setenv("PARAMETER_ENV_FOO", "not-bar")
  //

  // Get environment variables
  QUEUE_POLL_INTERVAL := getEnv("QUEUE_POLL_INTERVAL", "2")
  JOB_POLL_INTERVAL := getEnv("JOB_POLL_INTERVAL", "5")
  TIMEOUT := getEnv("TIMEOUT", "600")
  JENKINS_USER := getEnv("JENKINS_USER", "REQUIRED_ENV_VAR")
  JENKINS_TOKEN := getEnv("JENKINS_TOKEN", "REQUIRED_ENV_VAR")
  JENKINS_URI := getEnv("JENKINS_URI", "REQUIRED_ENV_VAR")
  JENKINS_JOB := getEnv("JENKINS_JOB", "REQUIRED_ENV_VAR")

  log.Info(fmt.Sprintf("Starting job: %s on Jenkins here: %s", JENKINS_JOB, JENKINS_URI))
  log.Debug(fmt.Sprintf("QUEUE_POLL_INTERVAL: %s JOB_POLL_INTERVAL: %s TIMEOUT: %s", QUEUE_POLL_INTERVAL, JOB_POLL_INTERVAL, TIMEOUT))

  query := url.Values{}
  for _, e := range os.Environ() {
    if strings.HasPrefix(e, "PARAMETER_") {
      pair := strings.Split(e, "=")
      query.Add(strings.TrimPrefix(pair[0], "PARAMETER_"), pair[1])
    }
  }
  log.Info("Query parameters: ")
  for key, value := range query {
    log.Info(fmt.Sprintf("%s=\"%s\"", key, value[0]))
  }

  //Submit the job request
  location, queue_id := postJob(JENKINS_USER, JENKINS_TOKEN, JENKINS_URI, JENKINS_JOB, query)
  //log.Debug(resp)
  log.Info("Request added at location: ", location)
  log.Debug("Job queue id: ", queue_id)

  //Poll queue for job start
  job_id := wait_for_start(JENKINS_USER, JENKINS_TOKEN, location, queue_id, QUEUE_POLL_INTERVAL, TIMEOUT)
  log.Info("Job started with id: ", job_id)

  //Poll job status until complete
  job_status := wait_for_complete(JENKINS_USER, JENKINS_TOKEN, JENKINS_URI, JENKINS_JOB, job_id, JOB_POLL_INTERVAL, TIMEOUT)
  log.Info(job_status)

}
