#!/usr/bin/env python3

import os, pathlib, sys

thisdir = os.path.abspath(os.path.dirname(sys.argv[0]))
godir = os.path.dirname(thisdir)
gogenpath = os.path.join(godir, 'go_generate.go')

abigenpath = 'go run ./generation/generate/wrap.go'

pkg_to_src = {}

for line in open(gogenpath):
    if abigenpath in line:
        abipath, pkgname = line.split(abigenpath)[-1].strip().split()
        srcpath = os.path.abspath(os.path.join(godir, abipath)).replace(
            '/abi/', '/src/').replace('.json', '.sol')
        if not os.path.exists(srcpath):
            srcpath = os.path.join(os.path.dirname(srcpath), 'dev',
                                   os.path.basename(srcpath))
        if not os.path.exists(srcpath):
            srcpath = srcpath.replace('/dev/', '/tests/')
        if os.path.basename(srcpath) != 'OffchainAggregator.sol':
            assert os.path.exists(srcpath), 'could not find ' + \
                os.path.basename(srcpath)
            pkg_to_src[pkgname] = srcpath

args = sys.argv[1:]

if len(args) == 0 or any(p not in pkg_to_src for p in args):
    print(__doc__.format(fastgen_dir=thisdir))
    print("Here is the list of packages you can build. (You can add more by")
    print("updating %s)" % gogenpath)
    print()
    longest = max(len(p) for p in pkg_to_src)
    colwidth = longest + 4
    header = "Package name".ljust(colwidth) + "Contract Source"
    print(header)
    print('-' * len(header))
    for pkgname, contractpath in pkg_to_src.items():
        print(pkgname.ljust(colwidth) + contractpath)
    sys.exit(1)

for pkgname in args:
    solidity_path = pkg_to_src[pkgname]
    outpath = os.path.abspath(os.path.join(godir, 'generated', pkgname,
                                           pkgname + '.go'))
    pathlib.Path(os.path.dirname(outpath)).mkdir(exist_ok=True)
    # assert not os.system(
    #     f'abigen -sol {solidity_path} -pkg {pkgname} -out {outpath}')
    cmd = f'abigen -sol {solidity_path} -pkg {pkgname} -out {outpath}'
    assert not os.system(cmd), 'Command "%s" failed' % cmd
