(module
    (import "env" "return_result" (func $return_result (param i32 i32 i32)))
    (memory $memory 1)
    (global $heap (mut i32) (i32.const 1024))
    (global $hello_size i32 (i32.const 7))
    (func $run_e (param $payload_ptr i32) (param $payload_size i32) (param $ident i32) (local $tmp1 i32) (local $tmp2 i32)
        (local.set $tmp1 ;; allocate some memory
            (call $allocate
                (global.get $hello_size)
            )
        )
        (memory.init 0 ;; load "Hello, " into that memory
            (local.get $tmp1)
            (i32.const 0)
            (global.get $hello_size)
        )
        (local.set $tmp2
            (call $concat ;; concat "Hello, " with the payload
                (local.get $tmp1)
                (global.get $hello_size)
                (local.get $payload_ptr)
                (local.get $payload_size)
            )
        )
        (call $return_result ;; return the concatenated string
            (local.get $tmp2)
            (i32.add
                (global.get $hello_size)
                (local.get $payload_size)
            )
            (local.get $ident)
        )
    )
    (func $concat (param $a_ptr i32) (param $a_size i32) (param $b_ptr i32) (param $b_size i32) (result i32) (local $tmp i32)
        (local.set $tmp
            (call $allocate
                (i32.add
                    (local.get $a_size)
                    (local.get $b_size)
                )
            )
        )
        (memory.copy
            (local.get $tmp)
            (local.get $a_ptr)
            (local.get $a_size)
        )
        (memory.copy
            (i32.add
                (local.get $tmp)
                (local.get $a_size)
            )
            (local.get $b_ptr)
            (local.get $b_size)
        )
        (local.get $tmp)
    )
    (func $allocate (param $size i32) (result i32)
        ;; round size to next multiple of 8 and advance the heap pointer
        global.get $heap
        global.get $heap
        local.get $size
        i32.const 7
        i32.add
        i32.const 0xfffffff8
        i32.and
        i32.add
        global.set $heap
    )
    (func $deallocate (param $ptr i32)
        ;; just leak all memory
        nop
    )
    (data "Hello, ")
    (export "run_e" (func $run_e))
    (export "allocate" (func $allocate))
    (export "deallocate" (func $deallocate))
    (export "memory" (memory $memory))
)
