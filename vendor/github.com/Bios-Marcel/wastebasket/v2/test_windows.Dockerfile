# nanoserver doesn't have a shell, so we can't run the test, as we require
# shell systemcalls such as SHFileOperationW.
FROM mcr.microsoft.com/windows/servercore:ltsc2022-amd64

WORKDIR /test
COPY wastebasket.test.exe ./test.exe
RUN test.exe

