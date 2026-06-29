# Changelog

## [0.23.0](https://github.com/googleapis/librarian/compare/v0.22.0...v0.23.0) (2026-06-29)


### Features

* **internal/librarian/java:** add decamelize utility for README generation ([#6538](https://github.com/googleapis/librarian/issues/6538)) ([a6d950a](https://github.com/googleapis/librarian/commit/a6d950a157338083086d2b3d8c03150179628b10)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** Add JSpecify dep to proto module template ([#6564](https://github.com/googleapis/librarian/issues/6564)) ([660ab2b](https://github.com/googleapis/librarian/commit/660ab2bcbb654ad546945b4a107fd77923ee57a9))
* **internal/librarian/java:** add production sample filter ([#6545](https://github.com/googleapis/librarian/issues/6545)) ([df213ec](https://github.com/googleapis/librarian/commit/df213ecb97834adb13c62b902fe090955845ba19)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** add README sample extraction helpers ([#6574](https://github.com/googleapis/librarian/issues/6574)) ([500ba22](https://github.com/googleapis/librarian/commit/500ba22a7de65f5ab0e84112daa7d16ff76ff841)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** add title override extractor ([#6546](https://github.com/googleapis/librarian/issues/6546)) ([0986d6a](https://github.com/googleapis/librarian/commit/0986d6a2cd5d3e4736b9e336a47dd85c9240e14b)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/python:** update gapic-generator to 1.36.0 ([#6548](https://github.com/googleapis/librarian/issues/6548)) ([a0930f4](https://github.com/googleapis/librarian/commit/a0930f4b0f52383ee4b19f23ae666ae4017f40f6))
* **nodejs:** bump pnpm version to 11.7.0 and support v11+ global bin layout ([#6494](https://github.com/googleapis/librarian/issues/6494)) ([1373628](https://github.com/googleapis/librarian/commit/1373628c9c8c639a666fe17813a4f313fe0f8c40)), closes [#6480](https://github.com/googleapis/librarian/issues/6480)
* **sidekick/rust:** track recording error info for discovery LROs ([#6304](https://github.com/googleapis/librarian/issues/6304)) ([feb2ef2](https://github.com/googleapis/librarian/commit/feb2ef22fa2b45f2bf216a7da95c106eeaacbad8)), closes [#6286](https://github.com/googleapis/librarian/issues/6286)
* **sidekick/swift:** generate synthetic messages ([#6530](https://github.com/googleapis/librarian/issues/6530)) ([7303648](https://github.com/googleapis/librarian/commit/730364823f86ac636ddea676bb439fce8f8e5a6c))
* **sidekick/swift:** handle clashing names ([#6543](https://github.com/googleapis/librarian/issues/6543)) ([56bf365](https://github.com/googleapis/librarian/commit/56bf365d10860f00bc9059a6d78f5f554c59406c))
* **sidekick/swift:** swap client vs. protocol ([#6566](https://github.com/googleapis/librarian/issues/6566)) ([14e186c](https://github.com/googleapis/librarian/commit/14e186cb2cdc1d8c545d022aba24bbd864e13ea0))


### Bug Fixes

* **internal/librarian:** gracefully handle nil defaults in applyDefaults ([#6571](https://github.com/googleapis/librarian/issues/6571)) ([3ee938a](https://github.com/googleapis/librarian/commit/3ee938a275e1f34480d3119331c8c97790f70aab))

## [0.22.0](https://github.com/googleapis/librarian/compare/v0.21.0...v0.22.0) (2026-06-22)


### Features

* **internal/librarian/java:** remove redundant keep items in librarian.yaml ([#6291](https://github.com/googleapis/librarian/issues/6291)) ([2965478](https://github.com/googleapis/librarian/commit/2965478f6348bdcdb53dc8ee9a142a1a0dfceac9))
* **internal/librarian/java:** support alternate_headers for monolithc libraries ([#6481](https://github.com/googleapis/librarian/issues/6481)) ([4165a09](https://github.com/googleapis/librarian/commit/4165a0982afee3c8ee28170b9c476eeeff2f2d17))
* **internal/librarian/nodejs:** add release level markdown generation ([#6476](https://github.com/googleapis/librarian/issues/6476)) ([1d1281f](https://github.com/googleapis/librarian/commit/1d1281f939b2b97df259c605fa70a7395626345e))
* **internal/librarian/nodejs:** add support for readme partials ([#6505](https://github.com/googleapis/librarian/issues/6505)) ([eca8e3d](https://github.com/googleapis/librarian/commit/eca8e3d26641893deea785e2d5850ace165ffee3)), closes [#6442](https://github.com/googleapis/librarian/issues/6442)
* **internal/librarian/nodejs:** extract sample metadata for node readme ([#6454](https://github.com/googleapis/librarian/issues/6454)) ([00e5e0d](https://github.com/googleapis/librarian/commit/00e5e0d8d0ff8307a861a8f5e08f8b896a1b8c50)), closes [#6442](https://github.com/googleapis/librarian/issues/6442)
* **internal/librarian/nodejs:** generate README in Node library ([#6520](https://github.com/googleapis/librarian/issues/6520)) ([68c0a20](https://github.com/googleapis/librarian/commit/68c0a2089d53f9f8604f23d86762afe26b66c884))
* **internal/librarian/nodejs:** implement README generation without partials ([#6488](https://github.com/googleapis/librarian/issues/6488)) ([44e6954](https://github.com/googleapis/librarian/commit/44e69540b74ad4f8b8dd212e0d97952d1af6814d))
* **internal/postprocessing:** implement Java method deprecation ([#6497](https://github.com/googleapis/librarian/issues/6497)) ([289a385](https://github.com/googleapis/librarian/commit/289a385ffb83092022fa8c98ec36bd90f4883dd7)), closes [#6298](https://github.com/googleapis/librarian/issues/6298)
* **internal/postprocessing:** implement Java method duplication ([#6484](https://github.com/googleapis/librarian/issues/6484)) ([0c5959c](https://github.com/googleapis/librarian/commit/0c5959c38e5c50805a7bd463c29a0351eb2e134c)), closes [#6298](https://github.com/googleapis/librarian/issues/6298)
* **nodejs:** support per-API version mixin configuration([#6462](https://github.com/googleapis/librarian/issues/6462)) ([71cd24e](https://github.com/googleapis/librarian/commit/71cd24eed5cc7bd8ac35493c597a493b3081e36d))
* **postprocessing:** implement Java method deletion ([#6436](https://github.com/googleapis/librarian/issues/6436)) ([820646f](https://github.com/googleapis/librarian/commit/820646f3db936c15e4efb8a7998fdc20201d70a2)), closes [#6298](https://github.com/googleapis/librarian/issues/6298)
* **sidekick/rust:** add condition to include `google-cloud-lro` as dependency ([#6503](https://github.com/googleapis/librarian/issues/6503)) ([7a89172](https://github.com/googleapis/librarian/commit/7a891726ea1062afa3641a40732948ca3800d199))
* **sidekick/rust:** bigquery query metadata ([#6407](https://github.com/googleapis/librarian/issues/6407)) ([6989ebc](https://github.com/googleapis/librarian/commit/6989ebcebf4cd2a04c36db7fcd544e8c464105b1))
* **sidekick/swift:** `bytes` for discovery docs ([#6433](https://github.com/googleapis/librarian/issues/6433)) ([d20f64c](https://github.com/googleapis/librarian/commit/d20f64c48ee852854344361eccaa60e5c76e9c58))
* **sidekick/swift:** generate method signature overloads ([#6473](https://github.com/googleapis/librarian/issues/6473)) ([27a72be](https://github.com/googleapis/librarian/commit/27a72beb520feb5d7c04701e967a3454f1367b39))
* **sidekick/swift:** qualified names for requests ([#6506](https://github.com/googleapis/librarian/issues/6506)) ([9489715](https://github.com/googleapis/librarian/commit/9489715246601f6f39268afbd4e4ad68240997b4))
* **sidekick:** parse method signatures ([#6451](https://github.com/googleapis/librarian/issues/6451)) ([7a433e7](https://github.com/googleapis/librarian/commit/7a433e71b701feda02bcf63e78624dcd08c5eb27))
* **sidekick:** parse method signatures ([#6461](https://github.com/googleapis/librarian/issues/6461)) ([16aa2e6](https://github.com/googleapis/librarian/commit/16aa2e6b1d00b63374d8571a0ba3e613b3768b28))


### Bug Fixes

* **internal/librarian/nodejs:** correct product doc link in readme template ([#6519](https://github.com/googleapis/librarian/issues/6519)) ([9cd8ee9](https://github.com/googleapis/librarian/commit/9cd8ee95b5be29f702dad03801c006e438448aa4)), closes [#6442](https://github.com/googleapis/librarian/issues/6442)
* **internal/librarian/nodejs:** path leak during generate_readme ([#6470](https://github.com/googleapis/librarian/issues/6470)) ([d3e7c16](https://github.com/googleapis/librarian/commit/d3e7c169c3e028720cd8d3c1369972bc376d2ede))
* **internal/postprocessing:** support deleting multiple methods and extract boundary finder ([#6471](https://github.com/googleapis/librarian/issues/6471)) ([20442d8](https://github.com/googleapis/librarian/commit/20442d805274eec9e1ab3362ad4586f3afe0957c)), closes [#6298](https://github.com/googleapis/librarian/issues/6298)
* **librarian:** print errors on failure ([#6458](https://github.com/googleapis/librarian/issues/6458)) ([37e4f91](https://github.com/googleapis/librarian/commit/37e4f915221045cba9e26f78c4e036d8d08076ed))
* **sidekick/rust:** disable docs/clippy warning for BQ generated files ([#6498](https://github.com/googleapis/librarian/issues/6498)) ([0a6a4d8](https://github.com/googleapis/librarian/commit/0a6a4d8f95b552a52d2d637d6db5f95499e5a9d8))
* **sidekick/rust:** use struct initializer for QueryMetadata ([#6504](https://github.com/googleapis/librarian/issues/6504)) ([2bdb3b5](https://github.com/googleapis/librarian/commit/2bdb3b5f262bad05ce2a569828f06b9445ab78bd))
* **sidekick/swift:** UrlSafe requires custom serialization ([#6522](https://github.com/googleapis/librarian/issues/6522)) ([09c74f6](https://github.com/googleapis/librarian/commit/09c74f696003106ccdfc104e8436b297a657b7bb))
* **surfer:** print errors on failure ([#6465](https://github.com/googleapis/librarian/issues/6465)) ([d91bf4c](https://github.com/googleapis/librarian/commit/d91bf4c4c6895fe7401cdea02fe0f2c64fb286d8))

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
