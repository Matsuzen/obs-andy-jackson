#!/bin/bash

sudo mv com.youtubestream.schedule.plist /Library/LaunchDaemons
sudo launchctl load /Library/LaunchAgents/com.youtubestream.schedule.plist
