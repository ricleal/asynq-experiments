# asynq-experiments
Experiments with Asynq: Simple, reliable &amp; efficient distributed task queue in Go

## exp1-simple

Simple stuff based on the webpage

## exp2-cron

Use Periodic Tasks

https://github.com/hibiken/asynq/wiki/Periodic-Tasks


## exp3-cron-advanced

Add and remove tasks: Dynamic Periodic Task

https://github.com/hibiken/asynq/wiki/Dynamic-Periodic-Task

- use redis to store pairs `<task-type> -> <cron-spec>`
- on a regular basis `GetConfigs()` and update the scheduler