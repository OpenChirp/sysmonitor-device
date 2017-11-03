# System Monitor OpenChirp Device
This device will report system statistics at a regular interval.
Currently, we report the following statistics/topics:
* `mem_total`
* `mem_avaliable`
* `mem_used`
* `mem_usedpercent`
* `disk_used`
* `disk_free`
* `disk_total`
* `disk_usedpercent`
* `load_1min`
* `load_5min`
* `load_15min`

# Operational Notes
* The interval can be changed dynamically through the `interval` topic of the
OC device.
* You can force a report by publishing any message to the `trigger` topic.

# Intervals
Interval durations must conform to the Golang time.ParseDuration string format
specified below:
```
A duration string is a possibly signed sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
```