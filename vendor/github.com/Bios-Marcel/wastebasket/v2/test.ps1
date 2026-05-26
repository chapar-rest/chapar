# remove preexisting test binaries from other systems to keep build context small
Remove-Item wastebasket.test*

# read the first arg or default to "windows"
$os = $args[0] ?? "windows"
$env:GOOS=$os; go test -c .
$env:DOCKER_CONTEXT="desktop-$os"; docker build -t temp_test -f ".\test_$os.Dockerfile" .
docker image rm temp_test

