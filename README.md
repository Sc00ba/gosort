# gosort

[![Go Test and Coverage](https://github.com/Sc00ba/gosort/actions/workflows/go-test-and-coverage.yml/badge.svg)](https://github.com/Sc00ba/gosort/actions/workflows/go-test-and-coverage.yml)
[![codecov](https://codecov.io/github/Sc00ba/gosort/branch/main/graph/badge.svg?token=OKOUED3X42)](https://codecov.io/github/Sc00ba/gosort)

## 💡About

This is a project I hack on from time to time with the goal of implementing the
GNU sort utility. It's more about the process and learning value than about
competing with GNU sort, but it's a fun exercise, and I'll continue to bring it
closer to feature and performance parity. For now, I hope you can learn
something from it, too.

![Architecture Diagram](docs/arch_1.png)

## Next Steps
- Decouple the writing of tmp files from the sorter
- Introduce a strategy for when the sorted data fits completely into memory
- Investigate using a sync pool to relieve GC pressure

## ⚖️ License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
