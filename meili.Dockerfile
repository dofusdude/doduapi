FROM getmeili/meilisearch

WORKDIR /meili_data

EXPOSE  7700/tcp

ENTRYPOINT ["tini", "--"]
CMD [ "/bin/meilisearch", "--enable-auto-batching", "--no-analytics", "--log-level", "ERROR" ]