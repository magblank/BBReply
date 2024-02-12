@echo off

echo Building bbreply-web...
cd bbreply-web
call npm install
call npm run build
cd ..

echo Copying dist to out\data...
xcopy /E /I /Y bbreply-web\dist out\data\dist
echo Contents of out\data:
dir out\data

echo Building Go program...
go build -o out\BBReply.exe .

echo Build process completed.
