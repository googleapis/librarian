FROM gcr.io/cloud-devrel-public-resources/java-library-generation:latest

COPY templates /workspace/templates
RUN chmod -R 777 /workspace

RUN mkdir /output
RUN chmod -R 777 /output

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh"]