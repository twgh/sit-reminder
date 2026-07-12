const fs = require('fs');
const path = require('path');

const ver = process.env.WINRES_VER;
const jsonPath = path.resolve(__dirname, '..', 'backend', 'winres', 'winres.json');

const j = JSON.parse(fs.readFileSync(jsonPath, 'utf8'));
j.RT_VERSION['#1']['0000'].fixed.file_version = ver;
j.RT_VERSION['#1']['0000'].fixed.product_version = ver;
j.RT_VERSION['#1']['0000'].info['0804'].FileVersion = ver;
j.RT_VERSION['#1']['0000'].info['0804'].ProductVersion = ver;

fs.writeFileSync(jsonPath, JSON.stringify(j, null, 2) + '\n', 'utf8');
