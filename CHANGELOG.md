# Changelog

## 0.4.2-mvp-dev

- Force UTF-8 console input, output, and pipeline encoding for Windows
  PowerShell 5.1 while retaining BOM-based script parsing.
- Preserve Unicode characters outside OEM code pages in captured output.
- Document the operator deployment path and prerequisite-free guest workflow.

## 0.4.1-mvp-dev

- Write structured PowerShell scripts with a UTF-8 BOM for Windows PowerShell
  5.1 compatibility.

## 0.4.0-mvp-dev

- Add Windows Job Object process-tree containment.
- Add structured PowerShell execution and CLIXML filtering.
- Add OEM, UTF-8, and UTF-16 output normalization.
- Add bounded chunked transfers and native ConPTY lifecycle primitives.
