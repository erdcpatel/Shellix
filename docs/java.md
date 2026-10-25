# Java Toolchain & Multi-Version Manager

Shellix provides built-in Java environment discovery, build automation, and **multi-JDK switching** across installed runtimes without requiring third-party plugins.

---

## 1. Features

- **Multi-JDK Version Discovery**: Scans macOS `/Library/Java/JavaVirtualMachines/`, SDKMAN (`~/.sdkman/candidates/java/`), Linux `/usr/lib/jvm/`, and Windows `Program Files` for all installed JDKs.
- **Instant Version Switching**: Switch between Java 8, 11, 17, 21, 25, or vendor flavors (Adoptium, Corretto, Zulu, Oracle) with a single command.
- **Subshell Isolation**: Launch a clean subshell with a target JDK active (`sx j shell 19`) without changing the global system configuration.
- **Automated Build Resolution**: Detects `pom.xml` (Maven) or `build.gradle` (Gradle) and prefers local wrappers (`./mvnw` or `./gradlew`) automatically.
- **Single-Source Execution**: Directly compile and execute standalone `.java` files via `sx java run`.

---

## 2. Multi-Version Management Commands

```bash
# List all discovered Java JDKs on the machine (with active version marked)
sx java list
# or alias
sx j ls

# Select a version and get export commands
sx j use 19

# Activate in current shell session immediately:
eval $(sx j use 19 -e)

# Launch an interactive subshell configured with that JDK:
sx j shell 19
```

---

## 3. Build & Run Commands

```bash
# Inspect current Java runtime and active project build tool
sx j env

# Trigger automated build using Maven or Gradle
sx j build

# Pass custom goals to the build tool
sx j build clean package -DskipTests

# Run a single Java file directly (Java 11+)
sx j run App.java
```

---

## 4. Multi-Version Builds in Workflows

You can target different Java versions for different steps in parallel:

```yaml
name: cross-version-build
steps:
  - name: build-legacy-service
    run: |
      eval $(sx j use 11 -e)
      ./mvnw clean compile
    parallel: true

  - name: build-modern-service
    run: |
      eval $(sx j use 21 -e)
      ./gradlew build
    parallel: true
```
