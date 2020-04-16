
target datalayout = "e-m:e-i64:64-f80:128-n8:16:32:64-S128"
target triple = "x86_64-unknown-linux-gnu"

@msg = constant [11 x i8] c"Hola Welt.\0A"
@stdout = external global i8*


define i8* @strPtrOf([11 x i8]* %global_str_ptr) {
b.2:
  %ret_ptr = getelementptr [11 x i8], [11 x i8]* %global_str_ptr, i32 0, i32 0
  ret i8* %ret_ptr
}

declare i64 @fwrite(i8*, i64, i64, i8*)
declare i16 @ferror(i8*)

define i16 @writeTo(i8* %str_ptr, i64 %str_len, i8* %out_file) {
begin:
  %n = call i64 @fwrite(i8* %str_ptr, i64 1, i64 %str_len, i8* %out_file)
  %ok = icmp eq i64 %n, %str_len
  br i1 %ok, label %end, label %err_case
err_case:
  %err_code = call i16 @ferror(i8* %out_file)
  br label %end
end:
  %ret_val = phi i16 [0, %begin], [%err_code, %err_case]
  ret i16 %ret_val
}

declare void @exit(i16)

define void @writeToStd(i8* %str_ptr, i64 %str_len, i8** %std_file) {
begin:
  %out_file = load i8*, i8** %std_file
  %err = call i16 @writeTo(i8* %str_ptr, i64 %str_len, i8* %out_file)
  switch i16 %err, label %end [i16 1, label %exit_on_err]
exit_on_err:
  call void @exit(i16 1)
  unreachable
end:
  ret void
}


define void @writeOut(i8* %str_ptr, i64 %str_len) {
b.3:
  call void @writeToStd(i8* %str_ptr, i64 %str_len, i8** @stdout)
  ret void
}


define i64 @main() {
b.1:
  %msg = call i8* @strPtrOf([11 x i8]* @msg)
  call void @writeOut(i8* %msg, i64 11)
  ret i64 0
}
