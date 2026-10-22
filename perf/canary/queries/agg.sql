SELECT count() AS count, type
FROM '__SOURCE__'
WHERE repo.name='NixOS/nixpkgs'
GROUP BY type
ORDER BY count DESC;
