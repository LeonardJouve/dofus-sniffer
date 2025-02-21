The sniffer still works but the dump_parser is unfortunatly useless since the new patch obfuscating enum values and adding useless fields to proto messages

To use dump_parser, dump the game using [Il2CppDumper](https://github.com/Perfare/Il2CppDumper) on `GameAssembly.dll` and `Dofus_Data\il2cpp_data\Metadata\global-metadata.dat` and move `dump.cs` at the root of this repository.
