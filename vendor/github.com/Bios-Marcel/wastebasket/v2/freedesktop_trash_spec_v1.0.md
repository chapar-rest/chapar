# The FreeDesktop.org Trash specification
Initial version written by Mikhail Ramendik <mr@ramendik.ru>
Content by David Faure <faure@kde.org>, Alexander Larsson <alexl@redhat.com>, Ryan Lortie <desrt@desrt.ca> and others on the FreeDesktop.org mailing list
Version 1.0

# Abstract

The purpose of this Specification is to provide a common way in which all “Trash can” implementations should store, list, and undelete trashed files. By complying with this Specification, various Trash implementations will be able to work with the same devices and use the same Trash storage. For example, if one implementation sends a file into the Trash can, another will be able to list it, undelete it, or clear it from the Trash.
Introduction

An ability to recover accidentally deleted files has become the de facto standard for today's desktop user experience.

Users do not expect that anything they delete is permanently gone. Instead, they are used to a “Trash can” metaphor. A deleted document ends up in a “Trash can”, and stays there at least for some time — until the can is manually or automatically cleaned.

This system has its own problems. Notably, cleaning disk space becomes a two-step operation — delete files and empty trash; this can lead to confusion for inexperienced users (“what's taking up my space?!”). Also, it is not easy to adapt the system to a multiuser environment. Besides, there is a potential for abuse by uneducated users — anecdotal evidence says they sometimes store important documents in the Trash can, and lose them when it gets cleaned!

However, the benefits of this system are so great, and the user expectation for it so high, that it definitely should be implemented on a free desktop system. And in fact, several implementations existed before this specification — some as command line utilities, some as preloaded libraries, and some as parts of major desktop environments. For example, both Gnome and KDE had their own trash mechanisms.

This Specification is to provide a common way in which all Trash can implementations must store trashed files. By complying with this Specification, various Trash implementations will be able to work with the same devices and use the same Trash storage.

This ability is important, at least, for shared network resources, removable devices, and in cases when different implementations are used on the same machine at different moments (for example, some users prefer Gnome, others prefer KDE, and yet others are command-line fans).
Scope and limitations

This Specification only describes the Trash storage. It does not limit the ways in which the actual implementations should operate, as long as they use the same Trash storage. Command line utilities, desktop-integrated solutions and preloaded libraries can work with this specification. 1

This Specification is geared towards the Unix file system tree approach. However, with slight modifications, it can easily be used with another kind of file system tree (for example, with drive letters).

A multi-user environment, where users have specific numeric identifiers, is essential for this Specification.

File systems and logon systems can be case-sensitive or non-case-sensitive; therefore, systems should generally not allow user names that differ only in case.

# Definitions

Trash, or Trash can — the storage of files that were trashed (“deleted”) by the user. These files can be listed, undeleted, or cleaned from the trash can.

Trashing — a “delete” operation in which files are transferred into the Trash can.

Erasing — an operation in which files (possibly already in the Trash can) are removed (unlinked) from the file system. An erased file is generally considered to be non-recoverable; the space used by this file is freed. [A “shredding” operation, physically overwriting the data, may or may not accompany an erasing operation; the question of shredding is beyond the scope of this document].

Original location — the name and location that a file (currently in the trash) had prior to getting trashed.

Original filename — the name that a file (currently in the trash) had prior to getting trashed.

Top directory , $topdir — the directory where a file system is mounted. “/” is the top directory for the root file system, but not for the other mounted file systems. For example, separate file systems might be mounted on “/home”, “/media/flash”, etc. In this text, the designation “$topdir” is used for “any top directory”.

User identifier , $uid — the numeric user identifier for a user. $uid is used here as “the numeric user identifier of the user who is currently logged on”.

Trash directory — a directory where trashed files, as well as the information on their original name/location and time of trashing, are stored. There may be several trash directories on one system; this Specification defines their location and contents. In this text, the designation “$trash” is used for “any trash directory”.

“Home trash” directory — a user's main trash directory. Its name and location is defined in this document.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

# Trash directories

A system can have one or more trash directories. The contents of any trash directory are to be compliant with the same standard, described below.

For every user a “home trash” directory MUST be available. Its name and location are $XDG_DATA_HOME/Trash 3; $XDG_DATA_HOME is the base directory for user-specific data, as defined in the Desktop Base Directory Specification .

The “home trash” SHOULD function as the user's main trash directory. Files that the user trashes from the same file system (device/partition) SHOULD be stored here (see the next section for the storage details). A “home trash” directory SHOULD be automatically created for any new user. If this directory is needed for a trashing operation but does not exist, the implementation SHOULD automatically create it, without any warnings or delays.

The implementation MAY also support trashing files from the rest of the system (including other partitions, shared network resources, and removable devices) into the “home trash” directory . This is a “failsafe” method: trashing works for all file locations, the user can not fill up any space except the home directory, and as other users generally do not have access to it, no security issues arise.

However, this solution leads to costly file copying (between partitions, over the network, from a removable device, etc.) A delay instead of a quick “delete” operation can be unpleasant to users.

An implementation MAY choose not to support trashing in some of these cases (notably on network resources and removable devices). This is what some well known operating systems do.

It MAY also choose to provide trashing in the “top directories” of some or all mounted resources. This trashing is done in two ways, described below as (1) and (2).

(1) An administrator can create an $topdir/.Trash directory. The permissions on this directories should permit all users who can trash files at all to write in it.; and the “sticky bit” in the permissions must be set, if the file system supports it.

When trashing a file from a non-home partition/device4 , an implementation (if it supports trashing in top directories) MUST check for the presence of $topdir/.Trash.

When preparing a list of all trashed files (for example, to show to the user), an implementation also MUST check for .Trash in all top directories that are known to it.

If this directory is present, the implementation MUST, by default, check for the “sticky bit”. (It MAY provide a way for the administrator, and only the administrator, to disable this checking for a particular top directory, in order to support file systems that do not have the “sticky bit”).

The implementation also MUST check that this directory is not a symbolic link.

If any of these checks fail, the implementation MUST NOT use this directory for either trashing or undeleting files, even if an appropriate $uid directory (see below) already exists in it. Besides, the implementation SHOULD report the failed check to the administrator, and MAY also report it to the user.

The following paragraph applies ONLY to the case when the implementation supports trashing in the top directory, and a $topdir/.Trash exists and has passed the checks:

If the directory exists and passes the checks, a subdirectory of the $topdir/.Trash directory is to be used as the user's trash directory for this partition/device. The name of this subdirectory is the numeric identifier of the current user ($topdir/.Trash/$uid). When trashing a file, if this directory does not exist for the current user, the implementation MUST immediately create it, without any warnings or delays for the user.

(2) If an $topdir/.Trash directory is absent, an $topdir/.Trash-$uid directory is to be used as the user's trash directory for this device/partition. $uid is the user's numeric identifier.

The following paragraph applies ONLY to the case when the implementation supports trashing in the top directory, and a $topdir/.Trash does not exist or has not passed the checks:

When trashing a file, if an $topdir/.Trash-$uid directory does not exist, the implementation MUST immediately create it, without any warnings or delays for the user.

When trashing a file, if this directory does not exist for the current user, the implementation MUST immediately create it, without any warnings or delays for the user.

Notes. If an implementation provides trashing in top directories at all, it MUST support both (1) and (2).

If an implementation does NOT provide trashing in top directories, and does provide the user with some interface to view and/or undelete trashed files, it SHOULD make a “best effort” to show files trashed in top directories (by both methods) to the user, among other trashed files or in a clearly accessible separate way.

When trashing a file, if the method (1) fails at any point — that is, the $topdir/.Trash directory does not exist, or it fails the checks, or the system refuses to create an $uid directory in it — the implementation MUST, by default, fall back to method (2), described below. Except for the case when $topdir/.Trash fails the checks, the fallback must be immediate, without any warnings or delays. The implementation MAY, however, provide a way for the administrator to disable (2) completely.

If both (1) and (2) fail (that is, no $topdir/.Trash directory exists, and an attempt to create $topdir/.Trash-$uid fails), the implementation MUST either trash the file into the user's “home trash” or refuse to trash it. The choice between these options can be pre-determined, or it can depend on the particular situation (for example, “no trashing of very large files”). However, if an implementation refuses to trash a file after a user action that generally causes trashing, it MUST clearly warn the user that the trashing has failed. It MUST NOT erase the file without user confirmation.

For showing trashed files, implementations SHOULD support (1) and (2) at the same time (that is, if both $topdir/.Trash/$uid and $topdir/.Trash-$uid are present, it should list trashed files from both of them).

# Contents of a trash directory

The previous section has described the location of trash directories. This section concerns the contents of any trash directory (including the “home trash” directory). This trash directory will be named “$trash” here.

A trash directory contains two subdirectories, named info and files.

The $trash/files directory contains the files and directories that were trashed. When a file or directory is trashed, it MUST be moved into this directory5 . The names of files in this directory are to be determined by the implementation; the only limitation is that they must be unique within the directory. Even if a file with the same name and location gets trashed many times, each subsequent trashing must not overwrite a previous copy. The access rights, access time, modification time and extended attributes (if any) for a file/directory in $trash/files SHOULD be the same as the file/directory had before getting trashed.

IMPORTANT NOTE. While an implementation may choose to base filenames in the $trash/files directory on the original filenames, this is never to be taken for granted6. A filename in the $trash/files directory MUST NEVER be used to recover the original filename; use the info file (see below) for that. (If an info file corresponding to a file/directory in $trash/files is not available, this is an emergency case, and MUST be clearly presented as such to the user or to the system administrator).

The $trash/info directory contains an “information file” for every file and directory in $trash/files. This file MUST have exactly the same name as the file or directory in $trash/files, plus the extension “.trashinfo”7.

The format of this file is similar to the format of a desktop entry file, as described in the Desktop Entry Specification . Its first line must be [Trash Info].

It also must have two lines that are key/value pairs as described in the Desktop Entry Specification:

    The key “Path” contains the original location of the file/directory, as either an absolute pathname (starting with the slash character “/”) or a relative pathname (starting with any other character). A relative pathname is to be from the directory in which the trash directory resides (for example, from $XDG_DATA_HOME for the “home trash” directory); it MUST not include a “..” directory, and for files not “under” that directory, absolute pathnames must be used. The system SHOULD support absolute pathnames only in the “home trash” directory, not in the directories under $topdir.

    The value type for this key is “string”; it SHOULD store the file name as the sequence of bytes produced by the file system, with characters escaped as in URLs (as defined by RFC 2396, section 2).

    The key “DeletionDate” contains the date and time when the file/directory was trashed. The date and time are to be in the YYYY-MM-DDThh:mm:ss format (see RFC 3339). The time zone should be the user's (or filesystem's) local time. The value type for this key is “string”.

Example:

[Trash Info]
Path=foo/bar/meow.bow-wow
DeletionDate=20040831T22:32:08

The implementation MUST ignore any other lines in this file, except the first line (must be [Trash Info]) and these two key/value pairs. If a string that starts with “Path=” or “DeletionDate=” occurs several times, the first occurence is to be used.8

Note that $trash/info has no subdirectories. For a directory in $trash/files, only an information file for its own name is needed. This is because, when a subdirectory gets trashed, it must be moved to $trash/files with its entire contents. The names of the files and directories within the directory MUST NOT be altered; the implementation also SHOULD preserve the access and modification time for them.

When trashing a file or directory, the implementation MUST create the corresponding file in $trash/info first. Moreover, it MUST try to do this in an atomic fashion, so that if two processes try to trash files with the same filename this will result in two different trash files. On Unix-line systems this is done by generating a filename, and then opening with O_EXCL. If that succeeds the creation was atomic (at least on the same machine), if it fails you need to pick another filename.
Directory size cache

In order to speed up the calculation of the total size of a particular trash directory, implementations (since version 1.0 of this specification) SHOULD create or update the $trash/directorysizes file, which is a cache of the sizes of the directories that were trashed into this trash directory. Individual trashed files are not present in this cache, since their size can be determined with a call to stat().

Each entry contains the name and size of the trashed directory, as well as the modification time of the corresponding trashinfo file (IMPORTANT: not the modification time of the directory itself)9.

The size is calculated as the disk space used by the directory and its contents, that is, the size of the blocks, in bytes (in the same way as the `du -B1` command calculates).

The modification time is stored as an integer, the number of seconds since Epoch. Implementations SHOULD use at least 64 bits for this number in memory.

The “directorysizes” file has a simple text-based format, where each line is:

[size] [mtime] [percent-encoded-directory-name]

Example:

16384 15803468 Documents
8192 15803582 Another_Folder

The last entry on each line is the name of the trashed directory, stored as the sequence of bytes produced by the file system, with characters escaped as in URLs (as defined by RFC 2396, section 2). Strictly speaking, percent-encoding is really only necessary for the newline character and for '%' itself. However, encoding all control characters or fully applying RFC 2396 for consistency with trashinfo files is perfectly valid, and even if an implementation does not use such encoding. it MUST be able to read names encoded with it.

The character '/' is not allowed in the directory name (even as %2F), since all these directories must be direct children of the "files" directory. Absolute paths are not allowed for the same reason.

To update the directorysizes file, implementations MUST use a temporary file followed by an atomic rename() operation, in order to avoid corruption due to two implementations writing to the file at the same time. The fact that the changes from one of the writers could get lost isn't an issue, as the cache can be updated again later on to add that entry.
Non-normative: suggested algorithm for calculating the size of a trash directory

load directorysizes file into memory as a hash directory_name -> (size, mtime, seen=false)
totalsize = 0
list "files" directory, and for each item:
    stat the item
    if a file:
        totalsize += file size
    if a directory:
        stat the trashinfo file to get its mtime
        lookup entry in hash
        if no entry found or entry's cached mtime != trashinfo's mtime:
            calculate directory size (from disk)
            totalsize += calculated size
            add/update entry in hash (size of directory, trashinfo's mtime, seen=true)
        else:
            totalsize += entry's cached size
            update entry in hash to set seen=true
done
remove entries from hash which have (seen == false)
write out hash back to directorysizes file

# Implementation notes

The names of the files/directories in $trash/info SHOULD be somehow related to original file names. This can help manual recovery in emergency cases (for example, if the corresponding info file is lost).

When trashing a file or directory, the implementation SHOULD check whether the user has the necessary permissions to delete it, before starting the trashing operation itself.

When copying, rather than moving, a file into the trash (when trashing to the “home trash” from a different partition), exact preservation of permissions might be impossible. Notably, a file/directory that was owned by another user will now be owned by this user (changing owners is usually only available to root). This SHOULD NOT cause the trashing operation to fail.

In this same situation, setting the permissions should be done after writing the copied file, as they might make it unwriteable..

A trashing operation might be refused because of insufficient permissions, even when the user does have the right to delete a file or directory. This may happen when the user has the right to delete a file/directory, but not to read it (or, in the case of a directory, to list it). In this case, the best solution is probably to warn the user, offering options to delete the file/directory or leave it alone. As noted earlier, when the user reasonably expects a file to be trashed, the implementation MUST NOT delete it without warning the user.

Automatic trash cleaning may, and probably eventually should, be implemented. But the implementation should be somehow known to the user.

If a directory was trashed in its entirety, it is easiest to undelete it or remove it from the trash only in its entirety as well, not as separate files. The user might not have the permissions to delete some files in it even while they do have the permission to delete the directory!

Important note on scope. This specification currently does NOT define trashing on remote machines where multiuser permissions are implemented but the numeric user ID is not supported, like FTP sites and CIFS shares. In systems implementing this specification, trashing of files from such machines is to be done only to the user's home trash directory (if at all). A future version may address this limitation.

# Administrativia

## Copyright and License

Copyright (C) 2004-2014 Mikhail Ramendik , mr@ramendik.ru .

The originators of the ideas that are described here did not object to this copyright. The author is ready to transfer the copyright to a standards body that would be committed to keeping this specification, or any successor to it, an open standard.

The license: Use and distribute as you wish. If you make a modified version and redistribute it, (a) keep the name of the author and contributors somewhere, and (b) indicate that this is a modified version.

Implementation under any license at all is explicitly allowed.

# Location

http://standards.freedesktop.org/trash-spec/trashspec-latest.html .

# Version history

0.1 “First try”, August 30, 2004. Initial draft. “Implementation notes” not written as yet.

0.2 August 30, 2004. Updated with feedback by Alexander Larsson <alexl@redhat.com> and by Dave Cridland <dave@cridland.net>

0.3 September 8, 2004. Changed the name and location of the “home trash” directory, and introduced the generic term “home trash”. Changed the trash info file format to a .desktop-like one. Added directions on creation of info files and copying of trashed files. Changed user names to user ids. Added implementation notes. Added a copyright notice.

0.4 September 9, 2004. Changed [Trash entry] to [Trash info] and fixed some typo's

0.5 September 9, 2004. Changed [Trash info] to [Trash Info]

0.6 October 8, 2004. Corrections by Alexander Larsson <alexl@redhat.com> . Also added “note on scope”. Cleaned up HTML. Added a link to this document on the freedesktop.org standards page

0.7 April 12, 2005. Added URL-style encoding for the name of the deleted file, as implemented in KDE 3.4

0.8 March 14, 2012. Update David Faure's email address, fix permanent URL for this spec.

1.0 January 2, 2014. Add directorysizes cache; style review.



1However, developers of preloaded libraries should somehow work around the case when a desktop environment also supporting the Trash specification is run on top of them. “Double trashing” and “trashing of the trash” should be avoided.

2To be more precise, for every user who can use the trash facility. In general, all human users, and possibly some “robotic” ones like ftp, should be able to use the trash facility.

3For case sensitive file systems, note the case.

4To be more precise, from a partition/device different from the one on which $XDG_DATA_HOME resides.

5“$trash/files/”, not into “$trash/” as in many existing implementations!

6At least because another implementation might trash files into the same trash directory

7For example, if the file in $trash/files is named foo.bar , the corresponding file in $trash/info MUST be named foo.bar.trashinfo

8This provides for future extension

9Rationale: if an older trash implementation restores a trashed directory, adds files to a nested subdir and trashes it again, the modification time of the directoy didn't change, so it is not a good indicator. However the modification time of the trashinfo file will have changed, since it is always the time of the actual trashing operation.
