
pushd containers/java
docker build -t generator/java .
popd

docker run \
  -v "$PWD/../google-cloud-java/generator-input/apigee-connect:/input" \
  -v "$PWD/tmp:/output" \
  -v "$PWD/../googleapis:/apis" \
  -t generator/java \
  generate \
  --api-root=/apis \
  --generator-input=/input \
  --output=/output