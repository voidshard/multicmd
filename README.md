#### What

Multicmd is a dead simple Golang cli tool that wraps ssh in order to run a given command on multiple hosts at a time & return all output to the caller.

#### Config

Multicmd looks for a config file using the env var MULTICMD_HOSTS. You'll want to set this.

The config file is pretty straight forward, it details a list of hosts like so:

```bash
ben@192.168.1.101:22:/home/ben/.ssh/my_id_file:london,master
ben@192.168.1.102:22:/home/ben/.ssh/my_id_file:paris,worker
ben@192.168.1.103:22:somepassword:oslo,worker
```

The format is thus:
```bash
user@host:port:login-credentials:tag,tag...
```

The login-credentials string here is either 
 - a filename (assumed to be an ssh key)
 - a password


The "tags" section at the end are user-specified strings that one can use to reference a host (or sets of hosts) to multicmd.
 - Note that "all" is a special tag that means .. well "all hosts" (surprise!) so there is no need to add a tag for this yourself.


When you invoke multicmd you pass in a tag like:
```bash
multicmd -t worker "sudo apt-get install -y something"
```
And multicmd will connect to both hosts & fire the command at them. (Check your commands before hitting enter, because this tool sure doesn't).

```bash
> multicmd -t mytag "uname -a"
[out] 192.168.1.101:22 Linux raspberrypi 4.14.98-v7+ #1200 SMP Tue Feb 12 20:27:48 GMT 2019 armv7l GNU/Linux
[out] 192.168.1.102:22 Linux raspberrypi 4.14.98-v7+ #1200 SMP Tue Feb 12 20:27:48 GMT 2019 armv7l GNU/Linux
```

Multicmd does it's best to catch user signals and send SIGABORT if the user signals the command to stop (eg. you ctrl-c frantically). 
Still exercise caution as this tool provides a very fast way of doing lots of damage.


#### About

Why? I have quite a few raspberry pis, switches, NAS etc at home and SSHing to all/subsets of them one at a time is tedious.
Maybe someone will find this useful .. or not ¯\_(ツ)_/¯

