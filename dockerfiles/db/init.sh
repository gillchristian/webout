#!/bin/sh -e

psql --variable=ON_ERROR_STOP=1 --username "postgres" <<-EOSQL
    CREATE ROLE webout WITH LOGIN PASSWORD 'webout';
    CREATE DATABASE "webout" OWNER = webout;
    GRANT ALL PRIVILEGES ON DATABASE "webout" TO webout;
EOSQL
