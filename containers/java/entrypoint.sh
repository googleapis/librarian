#!/usr/bin/env bash

die() { echo "$*" >&2; exit 2; }  # complain to STDERR and exit with error
needs_arg() { if [ -z "$OPTARG" ]; then die "No arg for --$OPT option"; fi; }

## Parse command
case "$1" in
    generate )    command="$1" ;;
    * )           die "Command not supported: $1" ;;
esac
shift

## Parse arguments
while getopts o:-: OPT; do  # Allow -o and long options, all requiring args
    # support long options: https://stackoverflow.com/a/28466267/519360
    if [ "$OPT" = "-" ]; then   # long option: reformulate OPT and OPTARG
        OPT="${OPTARG%%=*}"       # extract long option name
        OPTARG="${OPTARG#"$OPT"}" # extract long option argument (may be empty)
        OPTARG="${OPTARG#=}"      # if long option argument, remove assigning `=`
    fi
    case "$OPT" in
        api-root )          needs_arg; api_root="$OPTARG" ;;
        api-path )          needs_arg; api_path="$OPTARG" ;;
        generator-input )   needs_arg; generator_input="$OPTARG" ;;
        o | output )        needs_arg; output="$OPTARG" ;;
        repo-root )         needs_arg; repo_root="$OPTARG" ;;
        \? )        exit 2 ;;  # bad short option (error reported via getopts)
        * )         die "Illegal option --$OPT" ;;  # bad long option
    esac
done
shift $((OPTIND-1)) # remove parsed options and args from $@ list

if [[ -z "$api_root" ]]; then die "No value provided for --api-root"; fi;
if [[ -z "$generator_input" ]]; then die "No value provided for --generator-input"; fi;
if [[ -z "$output" ]]; then die "No value provided for --output"; fi;

echo "command=$command  api-root=$api_root  api-path=$api_path  generator-input=$generator_input  output=$output"

if find "$output/" -mindepth 1 -maxdepth 1 | read; then
   echo "Error: $output folder provided is not empty, cannot continue"
   exit 1
fi

log_file="$output/output.log"
library_name="apigee-connect"
workspace=/workspace

IS_MONO_REPO=true
if [[ "$IS_MONO_REPO" == "true" ]]; then
	library_workspace="$workspace/java-$library_name"
else
    die "Only mono repo supported at this time."
    library_workspace=$workspace
    echo "Processing non-monorepo, setting library workspace to $library_workspace"
fi

cp -r /input/* "$workspace/"

python /src/library_generation/cli/entry_point.py "$command" \
    --library-names="$library_name" \
    --repository-path="$repo_root" \
    --api-definitions-path="$api_root" \
    > "$log_file"

ls -la $workspace

if [ "$?" -ne 0 ]; then
    echo "Generation command failed for java/$library_name, check $log_file"
    exit 1
else
    echo "Generation command succeeded for java/$library_name"
  
    if [[ "$IS_MONO_REPO" == "true" ]]; then
        # the following lines undo the monorepo project structure, making each client library root into a standalone project
        # the maven projects will inherit from sdk-platform-java-config instead, similarly to other single-repo client libraries
        sed -i '/<parent>/,/<\/parent>/c \<parent><groupId>com.google.cloud</groupId><artifactId>sdk-platform-java-config</artifactId><version>3.40.0</version></parent>' "$library_workspace/pom.xml"
        sed -i '/<parent>/,/<\/parent>/c \<parent><groupId>com.google.cloud</groupId><artifactId>sdk-platform-java-config</artifactId><version>3.40.0</version></parent>' "$library_workspace/google-cloud-$library_name-bom/pom.xml"

        # because the parent has changed, shared-dependencies must be imported manually
        # TODO even before this is incorporated back into the generator, for code cleanliness, move this to a template
        sed -i '/<dependencyManagement>/,/<dependencies>/c \<dependencyManagement><dependencies><dependency><groupId>com.google.cloud</groupId><artifactId>google-cloud-shared-dependencies</artifactId><version>\${google-cloud-shared-dependencies.version}</version><type>pom</type><scope>import</scope></dependency><dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4.13.2</version><scope>test</scope></dependency>' "$library_workspace/pom.xml"

        # either duplicate these files, or add a skip config for the bom module, i.e:
        # https://github.com/googleapis/java-pubsub/blob/main/google-cloud-pubsub-bom/pom.xml#L70
        cp /workspace/templates/java.header "$library_workspace/google-cloud-$library_name-bom/"
        cp /workspace/templates/license-checks.xml "$library_workspace/google-cloud-$library_name-bom/"
    fi
	
    # each library needs a copy of these files in its root folder
    cp /workspace/templates/java.header "$library_workspace/"
    cp /workspace/templates/license-checks.xml "$library_workspace/"

  cp -a "$workspace/." /output/
fi
