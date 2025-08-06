# duola-sdk

```shell
export HTTP_PROXY=http://127.0.0.1:7899
export HTTPS_PROXY=http://127.0.0.1:7899
GOPROXY=https://proxy.golang.org go list -m github.com/fzxs8/duolasdk@v1.0.2-20250805172821

```

```shell

version="v1.0.3"

git tag -l $version
git tag -a $version -m "Release $version of duolasdk"
git push origin $version

git ls-remote --tags origin | grep $version
```