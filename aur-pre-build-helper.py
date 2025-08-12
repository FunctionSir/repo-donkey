import os
import sys

### CONF HERE ###
GitCloneDir = "/tmp"

if len(sys.argv) < 3:
    exit(1)

pkgName = sys.argv[1]
copyTo = sys.argv[2]
clonedDir = os.path.join(GitCloneDir, pkgName)

if not os.path.exists(clonedDir):
    ret = os.system("cd " + GitCloneDir + "; " +
                    "git clone https://aur.archlinux.org/"+pkgName+".git")
    if ret != 0:
        exit(ret)

ret = os.system("cd "+clonedDir+"; git pull origin master")
if ret != 0:
    exit(ret)

ret = os.system("cp -R --update=older "+clonedDir+"/* "+copyTo)
if ret != 0:
    exit(ret)
