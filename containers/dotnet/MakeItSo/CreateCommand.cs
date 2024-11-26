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

namespace MakeItSo;

internal class CreateCommand : ICommand
{
    private readonly string _apiRoot;
    private readonly string _api;
    private readonly string _outputRoot;
    private readonly UnknownApiBehavior _unknownApiBehavior;

    public CreateCommand(string apiRoot, string api, string outputRoot, UnknownApiBehavior unknownApiBehavior)
    {
        _apiRoot = apiRoot;
        _api = api;
        _outputRoot = outputRoot;
        _unknownApiBehavior = unknownApiBehavior;
    }

    public void Execute() => throw new NotImplementedException();
}
