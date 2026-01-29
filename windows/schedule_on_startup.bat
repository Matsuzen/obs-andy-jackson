@echo off

schtasks /create /tn "OBS-Youtube Stream Launcher" /tr "\"%~dp0..\launcher\launcher.exe\" stream schedule --time SUNRISE --city \"San Bernardino, CA\"" /sc onstart /rl HIGHEST /f >nul 2>&1
