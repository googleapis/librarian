// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License"):
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using System.Diagnostics;

namespace MakeItSo;

internal class UpdateCommand : ICommand
{
    private readonly string _apiRoot;
    private readonly string _api;
    private readonly string _outputRoot;
    private readonly UnknownApiBehavior _unknownApiBehavior;

    public UpdateCommand(string apiRoot, string api, string outputRoot, UnknownApiBehavior unknownApiBehavior)
    {
        _apiRoot = apiRoot;
        _api = api;
        _outputRoot = outputRoot;
        _unknownApiBehavior = unknownApiBehavior;
    }

    public void Execute()
    {
        MakeApiRootGitSafe();

        var apiCatalogJson = File.ReadAllText(Path.Combine(_outputRoot, "apis", "apis.json"));
        var apiCatalog = JsonConvert.DeserializeObject<ApiCatalog>(apiCatalogJson)!;
        var api = apiCatalog.Apis.FirstOrDefault(api => api.ProtoPath == _api);
        if (api is null)
        {
            switch (_unknownApiBehavior)
            {
                case UnknownApiBehavior.Create:
                    throw new NotImplementedException($"Create for unknown API {_api} is not yet supported");
                case UnknownApiBehavior.Error:
                    throw new InvalidOperationException($"No API configured with proto path {_api}, and unknown API behavior is 'error'");
                case UnknownApiBehavior.Ignore:
                    Console.WriteLine($"Ignoring unknown API {_api}");
                    return;
                default:
                    throw new InvalidOperationException($"Unsupported unknown API behavior: {_unknownApiBehavior}");
            }            
        }

        var psi = new ProcessStartInfo
        {
            FileName = "/bin/bash",
            ArgumentList = { "./generateapis.sh", api.Id },
            WorkingDirectory = _outputRoot,
            EnvironmentVariables = { { "GOOGLEAPIS_DIR", _apiRoot } }
        };
        var process = Process.Start(psi)!;
        process.WaitForExit();
        if (process.ExitCode != 0)
        {
            throw new Exception($"Generation ended with exit code {process.ExitCode}");
        }
    }

    /// <summary>
    /// Temporary hack to allow post-processors to reset any changes
    /// made in pre-processors. This will be obsolete once we stop the .NET generator container
    /// from issuing any git commands - possibly enforced by the CLI creating a copy of the API directory.
    /// </summary>
    private void MakeApiRootGitSafe()
    {
        var psi = new ProcessStartInfo
        {
            FileName = "/usr/bin/git",
            ArgumentList = { "config", "--global", "--add", "safe.directory", _apiRoot },
            WorkingDirectory = _outputRoot,
            EnvironmentVariables = { { "GOOGLEAPIS_DIR", _apiRoot } }
        };
        var process = Process.Start(psi)!;
        process.WaitForExit();
        if (process.ExitCode != 0)
        {
            throw new Exception($"git hack exited with code {process.ExitCode}");
        }
    }
}
