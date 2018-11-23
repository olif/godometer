# go-dometer
Measure pipe transfer speed. Inspired by the excellent [pv](http://www.ivarch.com/programs/pv.shtml "pv"). 

## Build
Build the application with:

    $> go build 

## Usage
Currently godometer can only measure transfer speed through a pipeline. Either
pipe data to godometer or use input redirection.
    
    $> gunzip -c indata.txt.gz | ./gm > out.txt

or

    $> ./gm < indata.txt.gz | gunzip > out.txt

<p align="center">
  <img src="../assets/screenshot.png?raw=true" alt="Screenshot"/>
</p>

# TODO
  + Implement support for multiprocess output

