# nanoserver doesn't have a shell, so we can't run the test, as we require
# shell systemcalls such as SHFileOperationW.
FROM scratch

# Required for os.CurrentUser
ENV USER=user
WORKDIR /test
COPY wastebasket.test /test_bin
ENTRYPOINT ["/test_bin"]

