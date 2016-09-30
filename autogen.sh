#/bin/bash
# Install hooks to automatically fmt on commit.

echo "Installing git hook scripts..."

ROOTDIR=$(pwd)

cat << EOF > .git/hooks/pre-commit
#!/bin/bash
# vet and check the style.
cd $ROOTDIR
make vet style
EOF
chmod +x .git/hooks/pre-commit

echo "Bootstrapped! If you move the project directory, you need to re-run this."
