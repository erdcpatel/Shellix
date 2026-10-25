# Java Toolchain & Build Integration

Shellix provides native Java environment discovery and build automation without forcing you to memorize different wrapper commands between projects.

---

## 1. Features

- **Automated JDK Discovery**: Identifies runtime version, compiler version (`javac`), and `JAVA_HOME`.
- **Build System Detection**: Scans the project root for `pom.xml` (Maven) or `build.gradle` / `build.gradle.kts` (Gradle).
- **Wrapper Auto-Resolution**: Prefers local wrappers (`./mvnw` or `./gradlew`) when present; falls back to system installations seamlessly.
- **Single-Source Execution**: Directly compiles and runs single `.java` files (Java 11+) via `sx java run`.

---

## 2. Commands

```bash
# Inspect the active Java runtime, compiler, and project build setup
sx java env
# or alias
sx j env

# Trigger automated build (runs ./mvnw clean compile or ./gradlew build)
sx j build

# Pass custom goals/flags to the underlying build tool
sx j build clean test -DskipTests

# Run a standalone Java file directly
sx j run App.java arg1 arg2
```

---

## 3. Parallel Java Builds in Shellix Flows

In multi-module or full-stack architectures, you can build your Java services in parallel with other frontend or Go modules:

```yaml
name: parallel-build-pipeline
steps:
  - name: build-backend-api
    run: sx j build -DskipTests
    parallel: true

  - name: build-frontend-client
    run: npm run build
    parallel: true
```
Both steps will execute concurrently, halving your local build time.
