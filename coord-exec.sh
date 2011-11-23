#!/bin/sh

#/ usage: exec.sh

did=$1
wid=0

[ -z "$did" ] && did=0

_set() {
  # TODO: use an id rather than 1
  printf 1 | doozer set /noeq/$did/$wid 0
}

while ! _set
do wid=`expr $wid + 1`
done

exec noeqd -w $wid -d $did
