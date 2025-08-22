/*
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

using System;
using System.Threading.Tasks;
using TestServerSdk;

// This program is just a thin wrapper around the installer logic in the SDK.
if (args.Length == 0)
{
    Console.WriteLine("Usage: installer <output_directory> [version]");
    return 1;
}

string outDir = args[0];
string version = args.Length > 1 ? args[1] : "v0.2.7";

await BinaryInstaller.EnsureBinaryAsync(outDir, version);
return 0;
