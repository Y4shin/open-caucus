UPDATE meetings SET secret = hex(randomblob(16)) WHERE secret = '' OR secret IS NULL;
