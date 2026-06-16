# Changelog

## [0.21.0](https://github.com/googleapis/librarian/compare/v0.20.0...v0.21.0) (2026-06-16)


### Features

* **internal/librarian/java:** source google-cloud-pom-parent in pom.xml templates ([#6432](https://github.com/googleapis/librarian/issues/6432)) ([c5718f4](https://github.com/googleapis/librarian/commit/c5718f4cba6bc602a4e2a35eca12e36083ba160c))
* **internal/librarian/java:** support alternate license header files ([#6311](https://github.com/googleapis/librarian/issues/6311)) ([e7222b1](https://github.com/googleapis/librarian/commit/e7222b17a11166851d0024227a93338623398014))
* **internal/librarian/nodejs:** add client_documentation_override to migrate ([#6310](https://github.com/googleapis/librarian/issues/6310)) ([cb8b040](https://github.com/googleapis/librarian/commit/cb8b040feda1f558b7c8284d679d2961aeaea814))
* **internal/librarian/python:** update gapic-generator to 1.35.0 ([#6427](https://github.com/googleapis/librarian/issues/6427)) ([c3a780e](https://github.com/googleapis/librarian/commit/c3a780e6f2c56b92f5882e43562e477608c071c8))
* **internal/librarian:** enable structured logging with slog ([#6363](https://github.com/googleapis/librarian/issues/6363)) ([458a738](https://github.com/googleapis/librarian/commit/458a738d51be0d1b85af70b8b2539d7c690f15c6)), closes [#6338](https://github.com/googleapis/librarian/issues/6338)
* **internal/postprocessing:** add copyFile function ([#6364](https://github.com/googleapis/librarian/issues/6364)) ([8aa57f0](https://github.com/googleapis/librarian/commit/8aa57f091c7b0952256f3679b34fab96d810b5c2)), closes [#6295](https://github.com/googleapis/librarian/issues/6295)
* **internal/postprocessing:** add removeFile function ([#6371](https://github.com/googleapis/librarian/issues/6371)) ([9e471eb](https://github.com/googleapis/librarian/commit/9e471eb6cfe6a60b1bd29e828d02bc2420b25b52)), closes [#6296](https://github.com/googleapis/librarian/issues/6296)
* **internal/postprocessing:** add replace and replaceRegex functions ([#6412](https://github.com/googleapis/librarian/issues/6412)) ([ece3aff](https://github.com/googleapis/librarian/commit/ece3aff3c68db01838314682e50d044ae1bb5329)), closes [#6297](https://github.com/googleapis/librarian/issues/6297)
* **librarian:** sync to release-please in add command ([#6346](https://github.com/googleapis/librarian/issues/6346)) ([f1103ae](https://github.com/googleapis/librarian/commit/f1103aea8d3b31a3de8e85d3f2e639ecf9acc9c8))
* **sidekick/rust:** add `gcp.resource.destination.id` and fix incorrect `gcp.longrunning.done` status in lro traces ([#6275](https://github.com/googleapis/librarian/issues/6275)) ([0648f55](https://github.com/googleapis/librarian/commit/0648f55b408e17c1c9daa19155d10ffd74222837))
* **sidekick/swift:** improve snippet body ([#6434](https://github.com/googleapis/librarian/issues/6434)) ([dcb6e6c](https://github.com/googleapis/librarian/commit/dcb6e6c0f73765cff8092cf41186ba0078ea413b))
* **sidekick/swift:** LRO snippets ([#6431](https://github.com/googleapis/librarian/issues/6431)) ([be95a09](https://github.com/googleapis/librarian/commit/be95a098262f83db28fa21ff0ec4ddf002afc649))


### Bug Fixes

* **.github/workflows:** fix outdated Java tools path in integration job ([#6372](https://github.com/googleapis/librarian/issues/6372)) ([72a5447](https://github.com/googleapis/librarian/commit/72a54479bae11e516ff0e5646f4a9ae3058bcb61))
* **golang:** fix onboarding versionless paths ([#6435](https://github.com/googleapis/librarian/issues/6435)) ([acd1c2b](https://github.com/googleapis/librarian/commit/acd1c2b6abbbac6e589acf2470927469a933b7e9))
* **internal/postprocessing:** return error for missing files in RemoveFile ([#6408](https://github.com/googleapis/librarian/issues/6408)) ([4a0e81b](https://github.com/googleapis/librarian/commit/4a0e81b040e598cb34896284ed016a6514335e2f))
* **librarian/internal/java:** preserve released_version for non-snapshot versions during tidy ([#6426](https://github.com/googleapis/librarian/issues/6426)) ([034374c](https://github.com/googleapis/librarian/commit/034374c2d0a37a62ee7c01e541ab8d555b3c9dcc))
* **sdk.yaml:** enable java sql v1beta4 dual transport ([#6437](https://github.com/googleapis/librarian/issues/6437)) ([ac320d3](https://github.com/googleapis/librarian/commit/ac320d388211ebc3c7387cad2b0fae590765e4c9))
* **sidekick/rust:** add clippy allow for BigQuery request methods ([#6373](https://github.com/googleapis/librarian/issues/6373)) ([cc804c9](https://github.com/googleapis/librarian/commit/cc804c95a750189ce065d745a379009f2de06c1a))

## [0.20.0](https://github.com/googleapis/librarian/compare/v0.19.0...v0.20.0) (2026-06-10)


### Features

* **nodejs:** add a DefaultVersion field to NodeJSPackage ([#6358](https://github.com/googleapis/librarian/issues/6358)) ([af3218f](https://github.com/googleapis/librarian/commit/af3218f8324be8bfaa0cff33afeea1c45d45a006))
* **sidekick/rust:** add bigquery code gen ([#6322](https://github.com/googleapis/librarian/issues/6322)) ([a7846f5](https://github.com/googleapis/librarian/commit/a7846f501eb2cece5813a13d600f69ae4d6e9897))
* **sidekick/swift:** non-string maps ([#6361](https://github.com/googleapis/librarian/issues/6361)) ([2b6d7e4](https://github.com/googleapis/librarian/commit/2b6d7e41f3db63a4be55f6a31e201c07edfc0b0b))
* **sidekick/swift:** support discovery-based modules ([#6351](https://github.com/googleapis/librarian/issues/6351)) ([09ef5cf](https://github.com/googleapis/librarian/commit/09ef5cf830158866b83c2eedcd2204dd6cdbe230))

## [0.19.0](https://github.com/googleapis/librarian/compare/v0.18.0...v0.19.0) (2026-06-09)


### Features

* **nodejs:** update tools for nodejs ([#6348](https://github.com/googleapis/librarian/issues/6348)) ([fdc4f18](https://github.com/googleapis/librarian/commit/fdc4f185c3a681c77128b5223403c9187e81036c))

## [0.18.0](https://github.com/googleapis/librarian/compare/v0.17.0...v0.18.0) (2026-06-09)


### Features

* **nodejs:** support client_documentation and client_documentation_override ([#6293](https://github.com/googleapis/librarian/issues/6293)) ([13919cc](https://github.com/googleapis/librarian/commit/13919ccd69ee04178476fe8c956e0de6c7dcc4d7))

## [0.17.0](https://github.com/googleapis/librarian/compare/v0.16.0...v0.17.0) (2026-06-09)


### Features

* **internal/cache:** add `BinDirectory` and `LIBRARIAN_BIN` override ([#6315](https://github.com/googleapis/librarian/issues/6315)) ([ac43e52](https://github.com/googleapis/librarian/commit/ac43e52b3a539e9ad574680fcc9ce88ab51d1728)), closes [#5850](https://github.com/googleapis/librarian/issues/5850) [#6199](https://github.com/googleapis/librarian/issues/6199)
* **librarian:** add `Discovery` field to Swift config ([#6320](https://github.com/googleapis/librarian/issues/6320)) ([2ee0a36](https://github.com/googleapis/librarian/commit/2ee0a363dbffd1c4d85ff70ac319577c0d45d0bf))
* **nodejs:** update gapic generator to v4.12.0 ([#6341](https://github.com/googleapis/librarian/issues/6341)) ([fae4158](https://github.com/googleapis/librarian/commit/fae4158f416fc2e6439aeb8b034199949942c9f5))
* **sidekick/rust:** use consolidated `LroRecorder` in tracing decorator ([#6259](https://github.com/googleapis/librarian/issues/6259)) ([0d318a9](https://github.com/googleapis/librarian/commit/0d318a96a131beb3f207654ff3dbb2de35cd95fb))
* **sidekick/swift:** generate `with` helper ([#6309](https://github.com/googleapis/librarian/issues/6309)) ([36d2aa1](https://github.com/googleapis/librarian/commit/36d2aa1217775c6d1a1df037c6e5cac9152a0831))
* **sidekick/swift:** map-based pagination ([#6268](https://github.com/googleapis/librarian/issues/6268)) ([082e996](https://github.com/googleapis/librarian/commit/082e996d1704bf9e4700441286d4834c83f97de7))


### Bug Fixes

* **internal/command:** look up executables in custom path environments ([#6273](https://github.com/googleapis/librarian/issues/6273)) ([7278ace](https://github.com/googleapis/librarian/commit/7278ace00162537372103588249295bde052c0e3)), closes [#6271](https://github.com/googleapis/librarian/issues/6271)
* **internal/fetch:** add support for symlink extraction ([#6321](https://github.com/googleapis/librarian/issues/6321)) ([7fa61e4](https://github.com/googleapis/librarian/commit/7fa61e4fad59c2833b0ae59b44f10240dd991ddf)), closes [#6313](https://github.com/googleapis/librarian/issues/6313)
* **internal/librarian/java:** allow omitting ReleasedVersion with fill and tidy ([#6274](https://github.com/googleapis/librarian/issues/6274)) ([9552dcd](https://github.com/googleapis/librarian/commit/9552dcdce156e4b4f24ab638eff01bcf69ce17d2)), closes [#6244](https://github.com/googleapis/librarian/issues/6244)
* **internal/librarian:** disable API path derive for Java ([#6287](https://github.com/googleapis/librarian/issues/6287)) ([bb3119f](https://github.com/googleapis/librarian/commit/bb3119f5a38464f912767222b188f829df4e8380))
* **librarian/internal/java:** explicitly list released_version as config ([5917f20](https://github.com/googleapis/librarian/commit/5917f20190fa9b3b8fd1af4ee5fc14eacd71c326))
* **librarian/swift:** configuration fields ([#6316](https://github.com/googleapis/librarian/issues/6316)) ([a1bd1c2](https://github.com/googleapis/librarian/commit/a1bd1c24eba7b3c073c9722d8041bf56b341d163))
* **nodejs:** manually create symlinks during librarian install ([#6314](https://github.com/googleapis/librarian/issues/6314)) ([bbdc773](https://github.com/googleapis/librarian/commit/bbdc773fa3eac516063c7ef72c2f5815275d6364)), closes [#6312](https://github.com/googleapis/librarian/issues/6312)
* **nodejs:** remove google/cloud/common_resources.proto after generation ([#6333](https://github.com/googleapis/librarian/issues/6333)) ([6a9e325](https://github.com/googleapis/librarian/commit/6a9e32542bdde60b27072977eb1a1d043d06fedf)), closes [#6024](https://github.com/googleapis/librarian/issues/6024)
* **python:** avoid adding to existing core lib ([#6324](https://github.com/googleapis/librarian/issues/6324)) ([9ebe312](https://github.com/googleapis/librarian/commit/9ebe31201f8d56fc1d916b6783306e6920f38d85))
* **sidekick/rust:** fix tracing template generation for discovery-based LROs ([#6258](https://github.com/googleapis/librarian/issues/6258)) ([33ef923](https://github.com/googleapis/librarian/commit/33ef923912bbf016b85eb32f00e7e09a852ddf59))
* **sidekick/swift:** warnings in snippets ([#6284](https://github.com/googleapis/librarian/issues/6284)) ([23bfa8d](https://github.com/googleapis/librarian/commit/23bfa8d0e9d6f5224527003ab9a1dbdadb37b25b))
