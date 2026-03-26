plugins {
    kotlin("jvm") version "1.9.22"
}

group = "com.google.testserver"
version = "0.1.0"

repositories {
    mavenCentral()
}

dependencies {
    implementation(kotlin("stdlib"))
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.0")
    
    // For JSON parsing (checksums.json)
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.3")

    // For YAML parsing (config files)
    implementation("org.yaml:snakeyaml:2.2")

    testImplementation("org.junit.jupiter:junit-jupiter:5.10.2")
}

tasks.test {
    useJUnitPlatform()
}

tasks.withType<org.jetbrains.kotlin.gradle.tasks.KotlinCompile>().configureEach {
    kotlinOptions {
        jvmTarget = "11"
    }
}
