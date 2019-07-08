# attiny-controller

This daemon communicates to the ATtiny microcontroller attached to
Cacophony Project Raspberry Pi based devices. It is responsible for
controller the device on/off window.

## DBUS API

attiny-controller exposes a simple API on the system DBUS. It listens
on `org.cacophony.ATtiny` and exposes objects its interface at
`/org/cacophony/ATtiny`. The following methods are available:

* `IsPresent() -> bool`: returns true if an ATtiny was detected.
* `StayOnFor(minutes)`: sets a number of minutes the device should
  stay on for (overiding any configured on/off window).

Here's an example of how to call the `IsPresent` API from the command line:

```
dbus-send \
    --system \
    --type=method_call \
    --print-reply \
    --dest=org.cacophony.ATtiny \
    /org/cacophony/ATtiny \
    org.cacophony.ATtiny.IsPresent
```

## Releases

Releases are built using TravisCI. To create a release visit the
[repository on Github](https://github.com/TheCacophonyProject/attiny-controllerreleases)
and then follow our [general instructions](https://docs.cacophony.org.nz/home/creating-releases)
for creating a release.
