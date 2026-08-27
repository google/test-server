/*
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

plugins {
  kotlin("jvm") version "1.9.22"
  kotlin("plugin.serialization") version "1.9.22"
  `maven-publish`
}

kotlin {
  jvmToolchain(17)
}

group = "com.google.testserver"

version = "0.1.0-SNAPSHOT"

repositories { mavenCentral() }

dependencies {
  implementation("org.yaml:snakeyaml:2.2")

  testImplementation(kotlin("test"))
  testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
  testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

tasks.test {
  useJUnitPlatform()
  testLogging {
    events("passed", "skipped", "failed", "standardOut", "standardError")
    showExceptions = true
    showCauses = true
    showStackTraces = true
    exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
  }
}

// This configuration is purely for integration testing purposes.
// It enables publishing the Kotlin SDK to the local maven cache (e.g. `./gradlew publishToMavenLocal`),
// so that local platform test suites (like `firebase-android-sdk`) can resolve and compile it.
publishing {
  publications {
    create<MavenPublication>("mavenJava") {
      from(components["java"])
    }
  }
}
