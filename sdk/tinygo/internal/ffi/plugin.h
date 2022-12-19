#include <stdint.h>

int32_t get_ffi_result(void* ptr, int32_t ident);
int32_t add_ffi_var(void* name_ptr, int32_t name_size, void* var_ptr, int32_t val_size, int32_t ident);

void return_result(void* rawdata, int32_t size, int32_t ident);
void return_error(int32_t code, void* rawdata, int32_t size, int32_t ident);

void log_msg(void *ptr, int32_t size, int32_t level, int32_t ident);

int32_t request_get_field(int32_t field_type, void* key_ptr, int32_t key_size, int32_t ident);
int32_t request_set_field(int32_t field_type, void *key_ptr, int32_t key_size, void *value_ptr, int32_t value_size, int32_t ident);

int32_t fetch_url(int32_t method, void *url_ptr, int32_t url_size, void *body_ptr, int32_t body_size, int32_t ident);

void resp_set_header(void *key_ptr, int32_t key_size, void *value_ptr, int32_t value_size, int32_t ident);
