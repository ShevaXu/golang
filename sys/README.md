# Sys

## Readdir

List and walk directory using system utils `ls`. This offer an alternative to stdlib's `filepath.Walk()`. Their behaviours are the same most of the time (read from `ls` is much slower though), but it's the rare cases that they are different make this alternative useful (on network devices differences are observed, so as during travis-CI build [broken](https://travis-ci.org/ShevaXu/golang/builds/328413613))

### TODO

- Investige why the difference (`filepath.Walk()` read system blocks directly);
- Determined test cases to see who is right;
- Benchmark the slow down.
