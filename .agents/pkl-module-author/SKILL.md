---
name: pkl-module-author
description: Write, review, test, and maintain Pkl configuration-as-code modules, templates, packages, generated bindings, and Pkl-backed infrastructure config. Use when editing .pkl files, PklProject, package definitions, pkl:test tests, Pkl codegen settings, or tools that use Pkl such as hk, pkl-k8s, pkl-pantry, and language bindings.
---

# Pkl Module Author

## Use this skill when

- Creating or modifying .pkl modules
- Designing reusable Pkl templates
- Writing PklProject
- Adding package dependencies
- Writing pkl:test tests
- Generating Go, Java, Kotlin, or Swift bindings
- Producing JSON, YAML, XML, plist, properties, or multi-file output
- Editing Pkl-backed tool configs such as hk.pkl
- Reviewing Pkl for validation, maintainability, or deployment safety

## Core rules

1. Treat Pkl as typed configuration-as-code, not YAML with functions.
2. Prefer templates for shared schema and amending modules for environment-specific values.
3. Use type annotations and constraints to catch invalid config before deployment.
4. Use amends when filling in an existing template.
5. Use extends only when deliberately extending an open module or class.
6. Keep reusable schema, defaults, and validation in template modules.
7. Keep environment or app-specific values in small amending modules.
8. Prefer generated output from pkl eval over hand-maintained JSON or YAML.
9. Add tests with pkl:test for important logic, constraints, and rendered examples.
10. For published packages, maintain apiTests to detect breaking changes.
11. Use package dependencies through PklProject, not ad hoc remote imports scattered everywhere.
12. For agents: always validate by running pkl eval, pkl test, or the consuming tool's dry-run command.

## First inspection workflow

When entering a repo, inspect the Pkl surface:

    find . -name '*.pkl' -o -name 'PklProject' -o -name 'PklProject.deps.json'
    pkl --version

Then run the obvious validation commands:

    pkl eval <module.pkl>
    pkl test
    pkl project resolve

If the repo uses generated bindings, also inspect:

    find . -name 'generator-settings.pkl' -o -name '*codegen*'
    find . -name go.mod -o -name build.gradle -o -name build.gradle.kts -o -name pom.xml

## Project layout guidance

Prefer this shape:

    pkl/
      templates/
        AppConfig.pkl
        Service.pkl
      env/
        dev.pkl
        staging.pkl
        prod.pkl
      tests/
        AppConfig.test.pkl
    PklProject
    PklProject.deps.json

For tool-specific config:

    hk.pkl
    PklProject

For generated bindings:

    pkl/
      AppConfig.pkl
    generator-settings.pkl
    go.mod / build.gradle.kts / pom.xml

## Template pattern

Use a template module for schema, defaults, and constraints:

    module AppConfig

    name: String(!isEmpty)

    port: UInt16(this > 0)

    environment: "dev"|"staging"|"prod"

    database: Database

    class Database {
      host: String(!isEmpty)
      port: UInt16(this > 0)
      username: String(!isEmpty)
      password: String(!isEmpty)
    }

Then use an amending module for a concrete environment:

    amends "../templates/AppConfig.pkl"

    name = "my-service"
    port = 8080
    environment = "dev"

    database {
      host = "localhost"
      port = 5432
      username = "app"
      password = read("env:DATABASE_PASSWORD")
    }

## Output rendering

Use CLI formats for simple output:

    pkl eval --format json pkl/env/dev.pkl
    pkl eval --format yaml pkl/env/dev.pkl

Use the module output property when output behavior is part of the module contract.

Example:

    output {
      renderer = new YamlRenderer {}
    }

For secrets, avoid rendering real values into committed files. Prefer read("env:NAME") or read("prop:NAME") and pass them at evaluation time.

## Testing pattern

Use pkl:test for assertions and golden examples.

    amends "pkl:test"

    import "../templates/AppConfig.pkl"

    facts {
      ["ports"] {
        8080.isBetween(1, 65535)
      }
    }

    examples {
      ["minimal config"] {
        new AppConfig {
          name = "demo"
          port = 8080
          environment = "dev"
          database {
            host = "localhost"
            port = 5432
            username = "demo"
            password = "example"
          }
        }
      }
    }

Run:

    pkl test

If expected example output changes intentionally, review the generated *-expected.pcf or *-actual.pcf before committing.

## PklProject pattern

Use PklProject to define dependencies, tests, evaluator settings, and package metadata.

    amends "pkl:Project"

    tests {
      ...import*("pkl/tests/**.pkl").keys
    }

    dependencies {
      // Add package dependencies here.
    }

After dependency changes:

    pkl project resolve

Commit both:

    PklProject
    PklProject.deps.json

## Package authoring checklist

For reusable packages, define:

- name
- baseUri
- version
- packageZipUrl
- description
- authors
- website
- documentation
- sourceCode
- license
- issueTracker
- apiTests

Use semantic versioning. Treat breaking schema changes as API changes.

## Codegen checklist

Before editing generated bindings:

1. Find the source .pkl schema.
2. Edit the schema first.
3. Run the language-specific generator.
4. Compile the consuming language.
5. Commit generated code only if the repo convention requires it.

For Go:

    pkl run @pkl.golang/gen.pkl pkl/AppConfig.pkl

For Java or Kotlin, check Gradle, Maven, or CLI setup before guessing flags.

## Security rules

- Do not hardcode secrets in .pkl.
- Prefer read("env:NAME") or read("prop:NAME").
- When evaluating untrusted Pkl, restrict module and resource access with CLI allowlists and root directory settings.
- Use --root-dir when file access must be constrained.
- Use --timeout for untrusted or expensive evaluation.
- Avoid broad remote imports in production config without pinning package versions.

## Agent workflow

When asked to change Pkl config:

1. Locate PklProject and relevant .pkl files.
2. Identify whether the target file is a template, amending module, generated config, or tool config.
3. Edit the smallest source module that owns the behavior.
4. Run:

    pkl eval <changed-module>
    pkl test

5. If the Pkl feeds a tool, run that tool's validation:

    hk check --all
    # or tool-specific dry run

6. Report exact commands run and any files changed.

## Diagnostic-driven development pattern

For diagnostic-driven workflows, prefer a small Pkl schema that declares the expected diagnostic before implementation.

Example:

    module WorkflowSpec

    name: String(!isEmpty)

    expectedDiagnostic: Diagnostic

    class Diagnostic {
      code: String(matches(Regex("[A-Z]{3}[0-9]{3}")))
      message: String(!isEmpty)
      file: String(!isEmpty)
    }

A deliberately red amending module may omit expectedDiagnostic so that pkl eval or pkl test surfaces the missing field.

Example diagnostic target:

    fixtures/red-workflow.pkl:7:3: error DDD001: workflow is missing expectedDiagnostic; diagnostic-driven workflows must declare the problem they expect to surface before implementation

## Review checklist

- Is this module a template or concrete config?
- Are required values typed?
- Are constraints explicit?
- Are environment-specific values isolated?
- Are repeated patterns abstracted cleanly?
- Are remote or package dependencies pinned?
- Does PklProject.deps.json match PklProject?
- Are tests present for logic and important examples?
- Does rendered output match the consuming tool's expected format?
- Are secrets kept out of source control?
- Did pkl eval and pkl test pass?

## Anti-patterns

Avoid:

- replacing Pkl with hand-written generated YAML
- putting all environments in one huge module
- hardcoding secrets
- scattering remote imports across many files
- editing generated language bindings without updating source .pkl
- adding abstractions before one concrete config works
- changing package APIs without apiTests
- skipping pkl eval after config edits

## Reporting format

When finished, report:

- files changed
- Pkl commands run
- rendered output target, if any
- test result
- any schema or package API changes
- exact next command the user should run
