"""Build rules for private GenAI SDK code."""

def _copy_from_filegroup_impl(ctx):
    """Rule implementation to copy files from a src_filegroup."""
    output_files = []
    src_files = ctx.attr.src_filegroup.files.to_list()

    for src_file in src_files:
        # Declare an output file in the current package with the same basename.
        out_file = ctx.actions.declare_file(src_file.basename)
        output_files.append(out_file)

        # Register an action to copy the file.
        # This creates a separate action for each file, which is generally better for caching.
        ctx.actions.run_shell(
            inputs = [src_file],
            outputs = [out_file],
            command = "cp -f %s %s" % (src_file.path, out_file.path),
            progress_message = "Copying %s to %s" % (src_file.short_path, out_file.short_path),
            mnemonic = "CopyFile",
        )

    # Return the generated files as default outputs.
    return [DefaultInfo(files = depset(output_files))]

copy_from_filegroup = rule(
    implementation = _copy_from_filegroup_impl,
    attrs = {
        "src_filegroup": attr.label(
            allow_files = True,  # Allows filegroup or other targets providing files
            mandatory = True,
            doc = "The filegroup target containing files to copy.",
        ),
    },
    doc = "Copies files from a filegroup in another package to the current package, preserving basenames.",
)
