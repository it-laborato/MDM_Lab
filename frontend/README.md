```sh
find . -type f \( -name "*.ts" -o -name "*.tsx" \) -exec sed -i '' -e 's/host/node/g' -e 's/Host/Node/g' {} +
find . -depth \( -type f -o -type d \) \( -name "*host*" -o -name "*Host*" \) -execdir bash -c '
  oldname="$(basename "$0")"
  newname="${oldname//host/node}"   # Replace all "host" with "node"
  newname="${newname//Host/Node}"   # Replace all "Host" with "Node"
  if [ "$oldname" != "$newname" ]; then
    mv -v -- "$oldname" "$newname"
  fi
' {} \;
```
