// Simple functions that build strings on the stack for hypervisor testing
// These functions intentionally return pointers to stack memory for testing purposes

// Manual string copy function to avoid external dependencies
void my_strcpy(char* dest, const char* src) {
    int i = 0;
    while (src[i] != '\0') {
        dest[i] = src[i];
        i++;
    }
    dest[i] = '\0';
}

// Simple function that builds a string on the stack and returns a pointer to it
// WARNING: This returns a pointer to stack memory, which is normally unsafe!
// But perfect for our hypervisor testing since we control the execution environment
char* build_stack_string() {
    char buffer[16];  // Local buffer on the stack
    
    // Build the string character by character
    buffer[0] = 'H';
    buffer[1] = 'e';
    buffer[2] = 'l';
    buffer[3] = 'l';
    buffer[4] = 'o';
    buffer[5] = ' ';
    buffer[6] = 'H';
    buffer[7] = 'V';
    buffer[8] = '!';
    buffer[9] = '\0';  // Null terminator
    
    // Return pointer to stack buffer (normally unsafe!)
    return buffer;
}

// Alternative version using our own strcpy
char* build_stack_string_strcpy() {
    char buffer[16];
    my_strcpy(buffer, "Hello HV!");
    return buffer;
}

// Version that builds a longer string with more stack usage
char* build_stack_string_long() {
    char buffer[32];  // Larger buffer to see more stack usage
    
    // Build the string manually to show stack operations clearly
    buffer[0] = 'S'; buffer[1] = 't'; buffer[2] = 'a'; buffer[3] = 'c';
    buffer[4] = 'k'; buffer[5] = ' '; buffer[6] = 'T'; buffer[7] = 'e';
    buffer[8] = 's'; buffer[9] = 't'; buffer[10] = ' '; buffer[11] = 'A';
    buffer[12] = 'R'; buffer[13] = 'M'; buffer[14] = '6'; buffer[15] = '4';
    buffer[16] = '\0';
    
    return buffer;
}

// Simple main function to make this a complete program
int main() {
    char* str = build_stack_string();
    // In a real program, we'd use the string here
    // For testing, we just return 0 for success
    return 0;
}