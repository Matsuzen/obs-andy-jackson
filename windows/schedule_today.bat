@echo off
:: OBS Stream Launcher
:: Schedules YouTube stream to start at sunrise-30min and end at sunset+30min

"%~dp0..\launcher\launcher.exe" stream schedule --time SUNRISE --city "San Bernardino, CA"
