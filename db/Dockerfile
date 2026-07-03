FROM postgres:16-alpine

ENV POSTGRES_DB=music_agent
ENV POSTGRES_USER=music_agent
ENV POSTGRES_PASSWORD=music_agent

COPY init.sql /docker-entrypoint-initdb.d/
