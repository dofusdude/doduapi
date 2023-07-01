FROM getmeili/meilisearch:v1.2.0

WORKDIR /meili_data

EXPOSE  7700/tcp

ENTRYPOINT ["tini", "--"]
CMD [ "/bin/meilisearch", "--no-analytics", "--log-level", "ERROR", "--env", "production" ]