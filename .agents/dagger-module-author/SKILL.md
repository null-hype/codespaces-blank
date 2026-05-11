---
name: dagger-module-author
description: Write, review, and maintain Dagger modules and agentic Dagger workflows. Use when creating Dagger Functions, module dependencies, toolchains, checks, Dagger LLM agents, or CI workflows driven by dagger call or dagger check.
---

# Dagger Module Author

## Use this skill when

- Creating or modifying a Dagger module
- Adding Dagger Functions, checks, toolchains, or module dependencies
- Building an agentic Dagger workflow with dag.LLM()
- Refactoring shell or CI scripts into Dagger
- Reviewing whether a Dagger module is reusable, testable, and agent-safe

## Core rules

1. Prefer small, composable Dagger Functions over one large workflow.
2. Give every exported function and important argument clear inline documentation.
3. Use typed inputs and outputs: Directory, File, Container, Secret, Service, custom objects.
4. Keep side effects explicit. Return artifacts rather than mutating the host directly.
5. Use Secret for credentials. Never pass secrets as plain strings.
6. Use CacheVolume for dependency caches, not ad hoc host mounts.
7. For agent workflows, expose a narrow tool surface.
8. Prefer a small workspace module with functions like readFile, writeFile, listFiles, and test.
9. Validate outputs by running tests inside the Dagger workflow before returning success.
10. Keep complex prompts in prompt files, not inline strings.
11. Document runnable examples with exact dagger call commands.

## Module creation workflow

1. Inspect existing dagger.json.
2. Identify the SDK: Go, TypeScript, Python, PHP, or Java.
3. Run or suggest these commands:

    dagger functions
    dagger call <function> --help

4. Add the smallest function that completes the workflow.
5. Add tests as Dagger Functions, preferably in a nearby tests or examples module.
6. Run:

    dagger develop
    dagger functions
    dagger call <new-function> ...
    dagger check

## Dependency rules

Add reusable modules with:

    dagger install github.com/org/repo/path@version

Prefer pinned tags or commits for reproducibility.

Use local module dependencies only when they live inside the same repo or module root.

After installing, use generated bindings through dag.

## Agent workflow pattern

For an agent that edits a codebase:

1. Create a workspace submodule.
2. Expose only the operations the agent needs:
   - read file
   - write file
   - list files
   - run tests
3. Create a main agent function that:
   - accepts an assignment
   - accepts source as a Directory
   - attaches the workspace to an Env
   - uses dag.LLM().WithEnv(...).WithPromptFile(...)
   - extracts the completed workspace or directory
   - runs tests again before returning
4. Return the modified Directory.

## Suggested structure

    dagger/
      main.go
      prompts/
        agent.md
      tests/
        module_test.go
    dagger.json

For a larger repo:

    dagger/
      main.go
      workspace/
        main.go
      prompts/
        implement.md
        review.md
      tests/
        main_test.go
    dagger.json

## Function design checklist

Before adding a function, answer:

- What is the smallest useful unit of work?
- What inputs should be typed?
- What output should be returned?
- What should be cached?
- What should be a Secret?
- What should be a Service?
- Can this run locally and in CI with the same command?
- Does the function need access to the whole source tree, or only a subdirectory?

## Secret handling rules

- Use Dagger Secret types for credentials.
- Do not accept tokens as plain strings.
- Do not echo secrets into logs.
- Do not write secrets into returned files or directories.
- Prefer explicit secret arguments over reading ambient host state.
- For test credentials, use short-lived or fake credentials where possible.

## Cache rules

- Use CacheVolume for package manager caches.
- Name cache volumes clearly.
- Do not rely on host-specific cache paths.
- Do not cache generated artifacts unless the cache boundary is clear.
- Make dependency installation deterministic before caching it.

## Testing rules

A Dagger module should have at least one of:

- a Dagger Function that acts as a check
- a language-native test
- a dagger check path
- a documented dagger call example that verifies behavior

For agent workflows, always test after agent output is produced.

## Review checklist

- Are function names discoverable via dagger functions?
- Are descriptions useful enough for humans and LLM tools?
- Are secrets typed as Secret?
- Are caches explicit?
- Are dependencies pinned?
- Can the function run locally and in CI with the same command?
- Is there a test or executable example?
- Does the module avoid unnecessary host access?
- For agents: is the tool surface narrow enough?
- For publishing: are examples, descriptions, and semver tags present?

## Anti-patterns

Avoid:

- one giant release() function that hides all behavior
- passing secrets as strings
- relying on unstated host tools
- mutating host files directly
- using broad source directories when a narrow directory is enough
- putting long prompts inline in code
- returning success before tests run
- adding generic plugin architecture before one concrete workflow exists

## Reporting format

When finished, report:

- files changed
- Dagger commands run
- test or check result
- any new function names
- exact next command the user should run
