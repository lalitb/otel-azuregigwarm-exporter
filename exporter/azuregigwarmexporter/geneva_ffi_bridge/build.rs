fn main() {
    // Set up rebuild triggers
    println!("cargo:rerun-if-changed=src/lib.rs");
}
