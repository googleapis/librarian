# Changelog

## [0.27.1](https://github.com/googleapis/librarian/compare/v0.27.0...v0.27.1) (2026-07-17)


### Bug Fixes

* **nodejs:** require cached tools and prevent host binary fallback ([#6874](https://github.com/googleapis/librarian/issues/6874)) ([599bde4](https://github.com/googleapis/librarian/commit/599bde4b85c6ca6bda5c694e29bb29b3a165a231))
* **nodejs:** unify pnpm v7 and v8+ global bin environment configuration ([#6873](https://github.com/googleapis/librarian/issues/6873)) ([ac5d9cc](https://github.com/googleapis/librarian/commit/ac5d9cc9bd1f5638e7c8d969338e6bd1789f96c5))

## [0.27.0](https://github.com/googleapis/librarian/compare/v0.26.0...v0.27.0) (2026-07-17)


### Features

* **internal/config:** add additional_protos configuration support for php ([#6801](https://github.com/googleapis/librarian/issues/6801)) ([26c1029](https://github.com/googleapis/librarian/commit/26c1029998095e3a4b9b0884b87c7adfb06c061c)), closes [#6743](https://github.com/googleapis/librarian/issues/6743)
* **internal/librarian/golang:** support protoc tool configuration ([#6767](https://github.com/googleapis/librarian/issues/6767)) ([7013350](https://github.com/googleapis/librarian/commit/7013350d238e507cbd2b8a2b108873ff7dd8d569))
* **internal/librarian/java:** populate RequiresBilling from API config in README template ([#6737](https://github.com/googleapis/librarian/issues/6737)) ([da3bc04](https://github.com/googleapis/librarian/commit/da3bc04ef2c4b1ff7eaf9da3d9f7a735baf51440))
* **internal/librarian/java:** pre-validate java libraries bom version in config ([#6674](https://github.com/googleapis/librarian/issues/6674)) ([620a505](https://github.com/googleapis/librarian/commit/620a505367f2f6141e8605668a7c42a9f7d7f8e9))
* **internal/librarian/nodejs:** support src_dir for pnpm tool sourcebuilds ([#6877](https://github.com/googleapis/librarian/issues/6877)) ([c58e52d](https://github.com/googleapis/librarian/commit/c58e52d4fb5bd2348e737f859556bcb124842371))
* **internal/librarian/php:** add tidying logic for additional_protos ([#6819](https://github.com/googleapis/librarian/issues/6819)) ([ec61444](https://github.com/googleapis/librarian/commit/ec61444c1fe8e2516240dcc0e756b06a264a9afc)), closes [#6743](https://github.com/googleapis/librarian/issues/6743)
* **internal/librarian/php:** install tools using composer ([#6799](https://github.com/googleapis/librarian/issues/6799)) ([482238f](https://github.com/googleapis/librarian/commit/482238f63fc23785324c11f6626a5b39676df7dc))
* **internal/librarian/php:** integrate additional_protos into client generation ([#6810](https://github.com/googleapis/librarian/issues/6810)) ([51bfdd2](https://github.com/googleapis/librarian/commit/51bfdd20e2dde592dbb71c7aedbf3a0144d49f8f)), closes [#6743](https://github.com/googleapis/librarian/issues/6743)
* **internal/librarian/php:** make CommonResources configurable for php ([#6823](https://github.com/googleapis/librarian/issues/6823)) ([f30d232](https://github.com/googleapis/librarian/commit/f30d232684436cded93b71b70d3f11bae0b830de)), closes [#6813](https://github.com/googleapis/librarian/issues/6813)
* **internal/librarian/php:** run owlbot.py in generate phase ([#6869](https://github.com/googleapis/librarian/issues/6869)) ([2d4b208](https://github.com/googleapis/librarian/commit/2d4b20874adbd2e66e691b122dcd3009fdbeca6e)), closes [#6773](https://github.com/googleapis/librarian/issues/6773)
* **internal/librarian/php:** support install PNPM tools ([#6839](https://github.com/googleapis/librarian/issues/6839)) ([192e469](https://github.com/googleapis/librarian/commit/192e469301a2bae0761ba0388923d6b823ac719e)), closes [#6830](https://github.com/googleapis/librarian/issues/6830)
* **librarian:** config get libraries apiPath ([#6655](https://github.com/googleapis/librarian/issues/6655)) ([e5cd774](https://github.com/googleapis/librarian/commit/e5cd774a50337bc2e9392d3704d9598dca2b457a))
* **sidekick/rust:** generate LRO `Poller` for bigquery `InsertJob` ([#6841](https://github.com/googleapis/librarian/issues/6841)) ([d5ce8a9](https://github.com/googleapis/librarian/commit/d5ce8a9d2faad987ad1ab1d61b54c915db03644d))
* **sidekick/swift:** `oneof` in method signatures ([#6863](https://github.com/googleapis/librarian/issues/6863)) ([9ebd4bc](https://github.com/googleapis/librarian/commit/9ebd4bc82f846af802c54896d2431e0b00b73ca6))
* **sidekick/swift:** generate core package versions ([#6812](https://github.com/googleapis/librarian/issues/6812)) ([1a08672](https://github.com/googleapis/librarian/commit/1a08672e0ccc004ac5d72a90a8be8ca0b2e38762)), closes [#5940](https://github.com/googleapis/librarian/issues/5940)
* **sidekick/swift:** generate telemetry headers ([#6820](https://github.com/googleapis/librarian/issues/6820)) ([b24d086](https://github.com/googleapis/librarian/commit/b24d086b214ba14df165d07481bbf92759c94537))
* **sidekick/swift:** use `$apiVersion` ([#6794](https://github.com/googleapis/librarian/issues/6794)) ([0b309b8](https://github.com/googleapis/librarian/commit/0b309b813e97f625d43e2e8efec314266460ce9d))
* **sidekick/swift:** use internal import over @_implementationOnly ([#6817](https://github.com/googleapis/librarian/issues/6817)) ([096d91c](https://github.com/googleapis/librarian/commit/096d91c81da87053cf4b11c2e77aec2476daa186))
* **swift:** add convert-swift functionality ([#6802](https://github.com/googleapis/librarian/issues/6802)) ([7813442](https://github.com/googleapis/librarian/commit/781344218cd1744fb46bdbc0ea0389f98564e7ec))
* **swift:** format all module output directories ([#6840](https://github.com/googleapis/librarian/issues/6840)) ([eb684c9](https://github.com/googleapis/librarian/commit/eb684c96123bdf925191694e1092ae80d9bb0723))
* **too/cmd/migrate:** specify protoc version in tools section ([#6793](https://github.com/googleapis/librarian/issues/6793)) ([201fa99](https://github.com/googleapis/librarian/commit/201fa99928ef26bd669e130089e1d2a8cf1b7546)), closes [#6791](https://github.com/googleapis/librarian/issues/6791)
* **tool/cmd/migrate:** add prettier formatting tools to PHP migration template ([#6831](https://github.com/googleapis/librarian/issues/6831)) ([9c06b20](https://github.com/googleapis/librarian/commit/9c06b20eeb454dda36fb8449ec61d8622708961c))
* **tool/cmd/migrate:** discover ruby libraries during migration ([#6856](https://github.com/googleapis/librarian/issues/6856)) ([fefcc0e](https://github.com/googleapis/librarian/commit/fefcc0eb387afed0412ad32311d90bfb1446c0ef)), closes [#6632](https://github.com/googleapis/librarian/issues/6632)
* **tool/cmd/migrate:** parse BUILD.bazel and populate AdditionalProtos for php ([#6814](https://github.com/googleapis/librarian/issues/6814)) ([53e3f4b](https://github.com/googleapis/librarian/commit/53e3f4b07a3732229582579590c97919c4bbe502)), closes [#6743](https://github.com/googleapis/librarian/issues/6743)
* **tool/cmd/migrate:** set php default for common_resources ([#6836](https://github.com/googleapis/librarian/issues/6836)) ([b2df615](https://github.com/googleapis/librarian/commit/b2df6153b76ce0b6ec71a563c20d3661941e19fa)), closes [#6813](https://github.com/googleapis/librarian/issues/6813)


### Bug Fixes

* **.github:** restore pnpm version ([#6851](https://github.com/googleapis/librarian/issues/6851)) ([0735cac](https://github.com/googleapis/librarian/commit/0735cac5691c433e449968a5479f4e1cfadb463b))
* **internal/config:** remove PHPPackage.AdditionalProtos and simplify logic ([#6818](https://github.com/googleapis/librarian/issues/6818)) ([19c4a93](https://github.com/googleapis/librarian/commit/19c4a9313aade80040bea86d3397f6fc71f82cab)), closes [#6743](https://github.com/googleapis/librarian/issues/6743)
* **internal/librarian/nodejs:** fallback to Checksum if SHA256 is empty ([#6804](https://github.com/googleapis/librarian/issues/6804)) ([cf906e6](https://github.com/googleapis/librarian/commit/cf906e68395726b82f7ad8be28329b4dee4ebd04)), closes [#6803](https://github.com/googleapis/librarian/issues/6803)
* **internal/librarian/nodejs:** pass --no-comments to compileProtos ([#6866](https://github.com/googleapis/librarian/issues/6866)) ([e749b3e](https://github.com/googleapis/librarian/commit/e749b3ed2e385f119f85443dc4dedd2e15db6ec3))
* **sidekick/swift:** clashes for `Logging` ([#6878](https://github.com/googleapis/librarian/issues/6878)) ([faf0c22](https://github.com/googleapis/librarian/commit/faf0c22f4a55d6cdfb675d8a464c2a99ed601084))
* **sidekick/swift:** dependencies on `Clients.swift` ([#6864](https://github.com/googleapis/librarian/issues/6864)) ([c5bca34](https://github.com/googleapis/librarian/commit/c5bca346e5b7ee074f09ec95e181fad38813b261))
* **sidekick/swift:** no escaping for `GateExpression` ([#6829](https://github.com/googleapis/librarian/issues/6829)) ([b26b680](https://github.com/googleapis/librarian/commit/b26b68022f12d8a4859de0b5fbfa325c9813c2e7))
* **sidekick/swift:** skip external package traits ([#6848](https://github.com/googleapis/librarian/issues/6848)) ([ba9041c](https://github.com/googleapis/librarian/commit/ba9041c6fe2739f14575c8b5ecd9f56adacbfef4))
* **tool/cmd/migrate:** add SHA256 for protoc version in php migrate ([#6798](https://github.com/googleapis/librarian/issues/6798)) ([0a47b2f](https://github.com/googleapis/librarian/commit/0a47b2f6b7f5202df79758b1e04184d57b31c59f))

## [0.26.0](https://github.com/googleapis/librarian/compare/v0.25.0...v0.26.0) (2026-07-13)


### Features

* **internal/gem:** add gem package to install Ruby gem tools ([#6724](https://github.com/googleapis/librarian/issues/6724)) ([40ba6df](https://github.com/googleapis/librarian/commit/40ba6dffd7c0d48792d7c1fdaac4fd7646881103))
* **internal/librarian/java:** add ApplyMoveActionsToLibrary helper and unit tests ([#6731](https://github.com/googleapis/librarian/issues/6731)) ([f25bd33](https://github.com/googleapis/librarian/commit/f25bd339cee587043ec4f7ef54478eadc7c5d9a9))
* **internal/librarian/java:** add RestructureToLibrary helper and unit tests ([#6757](https://github.com/googleapis/librarian/issues/6757)) ([b2ff68c](https://github.com/googleapis/librarian/commit/b2ff68c80cd763ae4eebd56fda0c8b40ca1200cc)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/librarian/java:** add ToKeepSet helper and unit tests ([#6730](https://github.com/googleapis/librarian/issues/6730)) ([df99304](https://github.com/googleapis/librarian/commit/df99304a051796b6808f3ed3c1d93be3d2644eed)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/librarian/java:** integrate native Go postprocessor into Java generator ([#6768](https://github.com/googleapis/librarian/issues/6768)) ([074059d](https://github.com/googleapis/librarian/commit/074059de0c7d06b417d42cf8ef975c19eac23f5b)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/librarian/java:** mark legacy postprocessing for deprecation ([#6716](https://github.com/googleapis/librarian/issues/6716)) ([78a4ab6](https://github.com/googleapis/librarian/commit/78a4ab65d9fff5b7e31f5069e0ee24c005921de7)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/librarian/nodejs:** use cache and bin directories for nodejs install ([#6680](https://github.com/googleapis/librarian/issues/6680)) ([7f88869](https://github.com/googleapis/librarian/commit/7f8886914595192f0c75eeeffd1d319faa2ed780))
* **internal/librarian/php:** add inital PHP client library generator ([#6703](https://github.com/googleapis/librarian/issues/6703)) ([9a45ab1](https://github.com/googleapis/librarian/commit/9a45ab13b3f65d5f655e4f1a531cd7906a1b10f1))
* **internal/librarian/php:** add tool installation directory helpers ([#6717](https://github.com/googleapis/librarian/issues/6717)) ([9cdf0b5](https://github.com/googleapis/librarian/commit/9cdf0b5aa31e1dadfb89481bc1ca8b440f2d6855)), closes [#6630](https://github.com/googleapis/librarian/issues/6630)
* **internal/librarian/ruby:** support installing Ruby gem dependencies ([#6751](https://github.com/googleapis/librarian/issues/6751)) ([bbce2c4](https://github.com/googleapis/librarian/commit/bbce2c444450041618c560d677cedb814b0ad517)), closes [#6634](https://github.com/googleapis/librarian/issues/6634)
* **internal/librarian:** add ruby tools directory to env output ([#6781](https://github.com/googleapis/librarian/issues/6781)) ([c220d71](https://github.com/googleapis/librarian/commit/c220d71060fb25e8fcb03f95046c19089ce8c39d))
* **internal/postprocessing:** add Apply pipeline runner and tests ([#6714](https://github.com/googleapis/librarian/issues/6714)) ([5cb8f66](https://github.com/googleapis/librarian/commit/5cb8f66b3fcc7d4e44e4f8e6a516e4eb29be3448)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/postprocessing:** add ApplyMethodOperations batch runner and tests ([#6698](https://github.com/googleapis/librarian/issues/6698)) ([8377d53](https://github.com/googleapis/librarian/commit/8377d530f1a2da8f36347fb2b1155220aa0f7c4d)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/postprocessing:** add applyToFiles and RemoveFiles ([#6673](https://github.com/googleapis/librarian/issues/6673)) ([2f4b437](https://github.com/googleapis/librarian/commit/2f4b43763d21de94bcd3c6601f7fe2ac10a76f7a)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/postprocessing:** add CopyFiles batch runner and tests ([#6686](https://github.com/googleapis/librarian/issues/6686)) ([20c3a1a](https://github.com/googleapis/librarian/commit/20c3a1a51727c8219335b83358544981f5da57f3))
* **internal/postprocessing:** add ReplaceAll and ReplaceRegexAll batch runners and tests ([#6688](https://github.com/googleapis/librarian/issues/6688)) ([bb18f06](https://github.com/googleapis/librarian/commit/bb18f06485b1b1f791f1a3a136e57e998e4185c5))
* **internal/protoc:** add protoc installation and use installed `protoc` in Java generation ([#6622](https://github.com/googleapis/librarian/issues/6622)) ([00ee24d](https://github.com/googleapis/librarian/commit/00ee24d1705d19b32fc49dc14df9d59c601c1ba6))
* **internal/protoc:** add Run function ([#6699](https://github.com/googleapis/librarian/issues/6699)) ([cd8a1d4](https://github.com/googleapis/librarian/commit/cd8a1d4bd53b8d5185e21c2a28f6897e755f28ac)), closes [#6558](https://github.com/googleapis/librarian/issues/6558)
* **internal/serviceconfig:** allowlist API paths for php ([#6789](https://github.com/googleapis/librarian/issues/6789)) ([7907686](https://github.com/googleapis/librarian/commit/790768673d7f7cfafbe53e19ab759810d6ef6dff)), closes [#6629](https://github.com/googleapis/librarian/issues/6629)
* **internal/tool/gem:** verify input directories and tools before installation ([#6778](https://github.com/googleapis/librarian/issues/6778)) ([3086d63](https://github.com/googleapis/librarian/commit/3086d632d01845d9e16f4701300d738406590a40))
* **java:** append versions.txt on add ([#6653](https://github.com/googleapis/librarian/issues/6653)) ([9e9e645](https://github.com/googleapis/librarian/commit/9e9e645c66706119ac7f89dacc7dc62a5ef9de92))
* **librarian/internal/config:** add php config ([#6701](https://github.com/googleapis/librarian/issues/6701)) ([939ae6f](https://github.com/googleapis/librarian/commit/939ae6f16e82cadcdd09aa111d7f71a0e81062da))
* **migrate:** discover and list PHP libraries during migration ([#6728](https://github.com/googleapis/librarian/issues/6728)) ([e552b89](https://github.com/googleapis/librarian/commit/e552b8998ead1cc36413abd660d4f4510c1b4ae7))
* **sidekick/parser:** correct LRO poller service ([#6704](https://github.com/googleapis/librarian/issues/6704)) ([1d4b2d1](https://github.com/googleapis/librarian/commit/1d4b2d14fedeeccfa3dd0f310c63cc249d5c1d7c))
* **sidekick/rust:** remove unstable gate for LRO tracing ([#6459](https://github.com/googleapis/librarian/issues/6459)) ([8266ce3](https://github.com/googleapis/librarian/commit/8266ce39cdfc9505a161b75110a6dd086d3739cb))
* **sidekick/swift:** discovery LROs ([#6738](https://github.com/googleapis/librarian/issues/6738)) ([c294d15](https://github.com/googleapis/librarian/commit/c294d15ed1d188375857f901c5f5d1dafdc6caea))
* **sidekick/swift:** generate deprecation attributes ([#6750](https://github.com/googleapis/librarian/issues/6750)) ([145e8ae](https://github.com/googleapis/librarian/commit/145e8aec839d6e728a50e1eb1b9c0b152f9ba138))
* **sidekick/swift:** traits with dependencies ([#6709](https://github.com/googleapis/librarian/issues/6709)) ([e8389f2](https://github.com/googleapis/librarian/commit/e8389f2a0461940be716856b720c215e9693c566))
* **swift:** add protobuf generation support ([#6697](https://github.com/googleapis/librarian/issues/6697)) ([3bebf26](https://github.com/googleapis/librarian/commit/3bebf26b420aa0b565d48164c24270b3b6d684d6))
* **tool/cmd/migrate/php:** scaffold composer tools for php ([#6736](https://github.com/googleapis/librarian/issues/6736)) ([f23b451](https://github.com/googleapis/librarian/commit/f23b451f39f022b3a8322d1577a0b04a289295ec))
* **tool/cmd/migrate:** add support to php ([#6726](https://github.com/googleapis/librarian/issues/6726)) ([586bac0](https://github.com/googleapis/librarian/commit/586bac0d5e5b4d7205e391ee4fbf9681144ac100)), closes [#6723](https://github.com/googleapis/librarian/issues/6723)
* **tool/cmd/migrate:** support union versions in PHP OwlBot configs ([#6782](https://github.com/googleapis/librarian/issues/6782)) ([7690206](https://github.com/googleapis/librarian/commit/76902068190a4e0566142b1e6fb03a35457aeae0)), closes [#6779](https://github.com/googleapis/librarian/issues/6779)


### Bug Fixes

* **internal/librarian/java:** remove excluded_poms from repometadata ([#6676](https://github.com/googleapis/librarian/issues/6676)) ([5e0f7f2](https://github.com/googleapis/librarian/commit/5e0f7f2e441ad1f913ffe037aad12ea0fa5e48a9))
* **internal/librarian/php:** enforce explicit API paths and add default output path ([#6740](https://github.com/googleapis/librarian/issues/6740)) ([347bbd6](https://github.com/googleapis/librarian/commit/347bbd6ba352ad4a29d75d591e2f26d060e88dcc))
* **internal/librarian:** preserve gem tools during tidy ([#6783](https://github.com/googleapis/librarian/issues/6783)) ([0150007](https://github.com/googleapis/librarian/commit/015000745c768d86290c04994d08416efc0521d1))
* **internal/librarian:** preserve maven and protoc configuration during tidy ([#6702](https://github.com/googleapis/librarian/issues/6702)) ([b439528](https://github.com/googleapis/librarian/commit/b4395280963cbd381125a8843cc94b75ed437e48)), closes [#6558](https://github.com/googleapis/librarian/issues/6558)
* **internal/snippetmetadata:** disable HTML escaping in JSON output ([#6777](https://github.com/googleapis/librarian/issues/6777)) ([4da5e28](https://github.com/googleapis/librarian/commit/4da5e283a04a91f64b993a65d0fb0155a063752a)), closes [#6776](https://github.com/googleapis/librarian/issues/6776)
* **librarian/rust:** detect inconsistent repos ([#6766](https://github.com/googleapis/librarian/issues/6766)) ([82ce5a9](https://github.com/googleapis/librarian/commit/82ce5a93457f1b47d8725f824e0af71d38dcd908))
* **sidekick/swift:** missing enum value docs ([#6727](https://github.com/googleapis/librarian/issues/6727)) ([e9020d2](https://github.com/googleapis/librarian/commit/e9020d2bf19886493aa74fa3618699072a1eaeb4))
* **tool/cmd/migrate:** populate API paths from .OwlBot.yaml during migrate for php ([#6739](https://github.com/googleapis/librarian/issues/6739)) ([6a15260](https://github.com/googleapis/librarian/commit/6a1526065a6739d6417bf97d429e9038d7c77757))

## [0.25.0](https://github.com/googleapis/librarian/compare/v0.24.0...v0.25.0) (2026-07-07)


### Features

* **internal/config:** add declarative postprocessing config schema ([#6651](https://github.com/googleapis/librarian/issues/6651)) ([12ecef3](https://github.com/googleapis/librarian/commit/12ecef3cc5ecf710f41b3af5b1c68bc15c1ddce0)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/filesystem:** add MoveAndMergeWithKeep and MoveAndMerge deprecation TODO ([#6644](https://github.com/googleapis/librarian/issues/6644)) ([fe91dda](https://github.com/googleapis/librarian/commit/fe91ddabce012d514ad171ae530ec7673a7fc06c)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **internal/librarian/java:** add comment to fully generated POM templates ([#6648](https://github.com/googleapis/librarian/issues/6648)) ([54f881b](https://github.com/googleapis/librarian/commit/54f881be2f540f0d48fafde876eb544bc901f9c4))
* **internal/librarian/java:** add Java README rendering without snippets ([#6636](https://github.com/googleapis/librarian/issues/6636)) ([96e890a](https://github.com/googleapis/librarian/commit/96e890a94817aaa004ea0eb1abed7afdcd4041a6)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** add logging-logback to legacy BOMs ([#6669](https://github.com/googleapis/librarian/issues/6669)) ([258d1de](https://github.com/googleapis/librarian/commit/258d1de0147d3d3d874015895dab778f96640e9e))
* **internal/librarian/java:** add README metadata parsing helpers ([#6595](https://github.com/googleapis/librarian/issues/6595)) ([c402d4a](https://github.com/googleapis/librarian/commit/c402d4a96213cdc8519924aecd4cb025eb5e268a)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** add README partials loader ([#6596](https://github.com/googleapis/librarian/issues/6596)) ([4babb38](https://github.com/googleapis/librarian/commit/4babb38e4e68913c0a2fdc872f477e4faf56f14c)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** pre-validate java libraries bom version in config ([#6613](https://github.com/googleapis/librarian/issues/6613)) ([6fe1ee3](https://github.com/googleapis/librarian/commit/6fe1ee3235dc4ad4ee39ed52c9fcdc85acf1417c))
* **internal/librarian/java:** source google-cloud-pom-parent in pom.xml templates ([#6607](https://github.com/googleapis/librarian/issues/6607)) ([3c46b73](https://github.com/googleapis/librarian/commit/3c46b735a8022b8272b0c6d7e73edb3ca2884b95))
* **internal/librarian/php:** add skeleton code for PHP client library generation ([#6641](https://github.com/googleapis/librarian/issues/6641)) ([dba27ec](https://github.com/googleapis/librarian/commit/dba27ecf21729abee39aca325f6310884dabe613))
* **internal/librarian:** add `protoc` installation support ([#6583](https://github.com/googleapis/librarian/issues/6583)) ([6037497](https://github.com/googleapis/librarian/commit/6037497ba994e88193d8d99458eb7ee3322893fa)), closes [#6558](https://github.com/googleapis/librarian/issues/6558)
* **internal/librarian:** track Go snippet metadata in release-please ([#6525](https://github.com/googleapis/librarian/issues/6525)) ([22c3b33](https://github.com/googleapis/librarian/commit/22c3b3371c30ac9b2bfd584ea4b02b7a19251bb6))
* **nodejs:** read nodejs tools from librarian.yaml config ([#6649](https://github.com/googleapis/librarian/issues/6649)) ([0bebff2](https://github.com/googleapis/librarian/commit/0bebff2b209bdbadca48216df52e91550430cce4))


### Bug Fixes

* **internal/postprocessing:** add input validation and error handling in Replace and ReplaceRegex ([#6661](https://github.com/googleapis/librarian/issues/6661)) ([1a32aab](https://github.com/googleapis/librarian/commit/1a32aabc20c0127e4a1f57a2f0d4554c7f68466b)), closes [#6516](https://github.com/googleapis/librarian/issues/6516)
* **rust:** do not add empty rust config ([#6609](https://github.com/googleapis/librarian/issues/6609)) ([c563304](https://github.com/googleapis/librarian/commit/c563304001e75ec6ebf7c9740c97afd012a7ea97))
* **rust:** tidy empty rust blocks ([#6618](https://github.com/googleapis/librarian/issues/6618)) ([dfa0f2d](https://github.com/googleapis/librarian/commit/dfa0f2da0dc61be993051444b7fafee6589a825e))

## [0.24.0](https://github.com/googleapis/librarian/compare/v0.23.0...v0.24.0) (2026-07-01)


### Features

* **add:** handle Release Please config for google-cloud-node ([#6569](https://github.com/googleapis/librarian/issues/6569)) ([1f8ee00](https://github.com/googleapis/librarian/commit/1f8ee00cfecad14cfb77b8b93d4e3b73b00100d5))
* **internal/librarian/java:** add code snippet extraction helpers for README rendering ([#6593](https://github.com/googleapis/librarian/issues/6593)) ([96f6925](https://github.com/googleapis/librarian/commit/96f6925fb95a81bdaa95e3405d4a09701a70178b)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/java:** add extractSamples for README generation ([#6578](https://github.com/googleapis/librarian/issues/6578)) ([b5e3d45](https://github.com/googleapis/librarian/commit/b5e3d4519d4a747fa8f09fdababb3bf77e78611c)), closes [#6515](https://github.com/googleapis/librarian/issues/6515)
* **internal/librarian/nodejs:** add metadata_name_override and name_pretty_override support ([#6603](https://github.com/googleapis/librarian/issues/6603)) ([3f6cfed](https://github.com/googleapis/librarian/commit/3f6cfed0b7d4c24c2d5aaa4750b4cc32db34cc51)), closes [#6453](https://github.com/googleapis/librarian/issues/6453)
* **internal/librarian:** add debug command with env subcommand ([#6576](https://github.com/googleapis/librarian/issues/6576)) ([027103b](https://github.com/googleapis/librarian/commit/027103b817777f101d72ad7df3af2bf7255e7b6a)), closes [#6374](https://github.com/googleapis/librarian/issues/6374)
* **internal/librarian:** populate Java Maven coordinates from defaults ([#6554](https://github.com/googleapis/librarian/issues/6554)) ([accb8ad](https://github.com/googleapis/librarian/commit/accb8adb99c8905d91489d59e6ed112159630874)), closes [#6513](https://github.com/googleapis/librarian/issues/6513)
* **librarian/swift:** use discovery config ([#6604](https://github.com/googleapis/librarian/issues/6604)) ([5a44ed7](https://github.com/googleapis/librarian/commit/5a44ed7b04f06fefcb1cf78c2e46b0f8bedeb035))
* **sidekick/discovery:** signatures without path params ([#6588](https://github.com/googleapis/librarian/issues/6588)) ([bb40e83](https://github.com/googleapis/librarian/commit/bb40e83a4e605ac0cc9b1719f8d965b4a68bb908))


### Bug Fixes

* **internal/librarian/java:** exclude google-cloud-bom and libraries-bom when generating gapic-libraries-bom/pom.xml ([#6601](https://github.com/googleapis/librarian/issues/6601)) ([b8e50a5](https://github.com/googleapis/librarian/commit/b8e50a51a11cb07192f209c0e7fadeb5f3ce77c7))
* **internal/serviceconfig:** normalize transport name for Java repo-metadata ([#6582](https://github.com/googleapis/librarian/issues/6582)) ([e20f77a](https://github.com/googleapis/librarian/commit/e20f77a70df502836fedda0f302fda66b09f1452))
* **sdk.yaml:** allow rust for many non-cloud apis ([#6598](https://github.com/googleapis/librarian/issues/6598)) ([0efb6e7](https://github.com/googleapis/librarian/commit/0efb6e71ddbdb5fd6d121a02ecbf02138235e102))

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
