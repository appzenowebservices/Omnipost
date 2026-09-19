#!/bin/bash

pkill -9 patra
 cd ../
./patra --install --yes
./patra > /dev/null 2>/dev/null &
