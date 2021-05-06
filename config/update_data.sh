#!/bin/sh

APP_LBL='diff-privacy-beam'
ETC_PATH="/etc/${APP_LBL}"  # app config info, scripts, ML models, etc.

echo "setting resource limits for cron"
ulimit -e 0
ulimit -i 31854
ulimit -p 8
ulimit -q 819200

echo "initializing languages in databases"
${ETC_PATH}/resources/init_db

echo "getting rid of old synthetic data"
${ETC_PATH}/resources/clean_db data

echo "conducting private and public counts of entries in database using apache beam"
${ETC_PATH}/resources/beam

echo "getting rid of old days of output counts"
${ETC_PATH}/resources/clean_db output