# Psstat (a Telegraf exec)

Psstat gathers resource usage information of specified processes. The output
of psstat can be fed into [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/)
as influx format. psstat is an alternative to the build in [procstat](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/procstat)
of Telegraf itself. The main differences are: (1.) only a few fields are
outputted, greatly reducing the data pushed to Influxdb; (2.) resource usage of
child processes is calculated as well giving you a better insight in the
resource usage of a process.

psstat was build for speed and a low impact on the host machine. So data is
gathered from `/proc` only. As written above, only a few fields are gathered
and shown in the output. These fields are CPU usage (percentage), memory usage
(percentage) and number of processes. More detailed explanation of the output
can be found below at [measurements](#measurements).

A cache to calculate the difference between measurements is kept in a directory
of your choice, however, I advise to make this a `tmpfs` disk to keep it fast.
See [cache on tmpfs](#cache-on-tmpfs) for instructions.

If you run multiple exec's for psstat you should give each exec its own
cache-name to prevent data corruption in the cache. If this becomes corrupt the
output will be empty most likely.

Processes to monitor can be specified in different ways:

- By PID number
- By PID file
- By pattern (matched in `/proc/<id>/stat`)
- By Systemd Unit pattern

The first run of the psstat won't return any result as that run is required
to build the first cache. Consecutive runs will output data.

## Example usage in Telegraf

```
[[inputs.exec]]
  commands = ["/usr/bin/psstat --tag env=production --pid-file /var/run/nginx.pid --pattern mysql:mysqld_safe"]
  data_format = "influx"
```

The above configuration would result in output like:
```
> psstat,host=tengu,env=production,process_name=nginx p_cpu=0.42,p_mem=0.001,n_proc=5i 1562471764000000000
> psstat,host=tengu,env=production,process_name=mysql p_cpu=3.14,p_mem=1.592,n_proc=65i 1562471764000000000
```

## Cache on tmpfs

To keep the cache fast you can store it on a `tmpfs` disk. By default psstat
tries to write cache data to `/tmp` which should be a `tmpfs` already.

If you want a dedicated directory for the cache of psstat (for example in a
multi tenant environment) you can do the following:

Assumption: Telegraf is used which runs as the user `telegraf`. If not, adjust
the user and group in the commands below.

Add the following line to `/etc/fstab`:

```shell
tmpfs /mnt/psstat tmpfs auto,rw,mode=755,size=5m,uid=telegraf,gid=telegraf 0 2
```

This creates a disk of 5 MB to store caches.

Then you need to create the directory specified above and mount all devices:

```shell
mkdir /mnt/psstat
chown telegraf: /mnt/psstat
mount -a
```

## Measurements

CPU usage is measured as a percentage where 100% is written as 1.000. If a
value greater than 1.000 is shown, it means more than 1 core is used for that
process (and its children).

- psstat.p_cpu value=3.141

Memory usage is measured as a percentage between 0.000 and 1.000.

- psstat.p_mem value=0.042

Number of processes, integer. This is the count of the process and its children.

- psstat.n_proc value=3
