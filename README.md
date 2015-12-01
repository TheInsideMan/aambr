Apps API Messaging Board Relay (AAMBR)
===============

## Introduction

Takes data from Graphite and summerizes it. This summerized data will eventually be used as part of a Raspberry Pi LCD project. The Graphite data originates from the Apps API. At the moment this repo only deals with response codes but it would be simple to extent this to other data sets such as CPU usage, Memory, Average responce times etc.

## Requirements

 - Go 1.5
 - All vendor packages are contained within this repo. Once you use the "$ ./build.sh" file to build a binary all should be good. 
 - Create a folder called "config" and in it a file called "dev.json" this should contain the following information...

```
 {
  "graphite": {
      "un": "username",
      "pw": "your_password_goes_here"
  }
}
```