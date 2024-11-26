// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License").
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at 
//
// https://www.apache.org/licenses/LICENSE-2.0 
//
// Unless required by applicable law or agreed to in writing, software 
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and 
// limitations under the License.


// Quick and dirty prototype for an entry point for containerized generation.

using MakeItSo;

var namedArgs = new Dictionary<string, string>();

foreach (var arg in args)
{
    if (!arg.StartsWith("--"))
    {
        Console.WriteLine($"Invalid argument: {arg}");
        return 1;
    }
    else
    {
        var split = arg.Split('=', 2);
        if (split.Length != 2)
        {
            Console.WriteLine($"Invalid argument: {arg}");
            return 1;
        }
        namedArgs[split[0][2..]] = split[1];
    }
}

var commandName = namedArgs.GetValueOrDefault("command");
if (commandName is null)
{
    Console.WriteLine("No command specified.");
    ShowHelp();
    return 1;
}

UnknownApiBehavior unknownApiBehavior = namedArgs.GetValueOrDefault("unknown", "error") switch
{
    "error" => UnknownApiBehavior.Error,
    "ignore" => UnknownApiBehavior.Ignore,
    "create" => UnknownApiBehavior.Create,
    _ => throw new ArgumentException($"Invalid value for --unknown: '{namedArgs["unknown"]}'")
};

ICommand? command;
try
{
    command = namedArgs["command"] switch
    {
        "create" => new CreateCommand(namedArgs["api-root"], namedArgs["api"], namedArgs["output-root"], unknownApiBehavior),
        "update" => new UpdateCommand(namedArgs["api-root"], namedArgs["api"], namedArgs["output-root"], unknownApiBehavior),
        "build" => new BuildCommand(namedArgs["api"], namedArgs["output-root"], unknownApiBehavior),
        _ => null
    };
}
catch (KeyNotFoundException ex)
{
    Console.WriteLine(ex.Message);
    ShowHelp();
    return 1;
}

if (command is null)
{
    Console.WriteLine($"Unknown command: {namedArgs["command"]}");
    ShowHelp();
    return 1;
}

try
{
    command.Execute();
}
catch (Exception e)
{
    Console.WriteLine($"Error executing command: {e}");
    return 1;
}

return 0;

void ShowHelp()
{
    Console.WriteLine("Valid commands:");
    Console.WriteLine("--command=update --api-root=path-to-apis --api=relative-path-to-api --output-root=path-to-dotnet-repo");
    Console.WriteLine("--command=create --api-root=path-to-apis --api=relative-path-to-api --output-root=path-to-dotnet-repo");
    Console.WriteLine("--command=build --api=relative-path-to-api --output-root=path-to-dotnet-repo");
    Console.WriteLine();
    Console.WriteLine("Common optional arguments:");
    Console.WriteLine("--unknown=error|ignore|create");
}
