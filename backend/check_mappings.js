const { common } = require('mavlink-mappings');
console.log('GlobalPositionInt:', !!common.GlobalPositionInt);
console.log('Attitude:', !!common.Attitude);
console.log('SysStatus:', !!common.SysStatus);
console.log('GpsRawInt:', !!common.GpsRawInt);
console.log('VfrHud:', !!common.VfrHud);
console.log('StatusText:', !!common.StatusText);
