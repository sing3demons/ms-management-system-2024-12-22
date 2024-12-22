const { authFunction } = require('./index');

test('hello world!', () => {
	expect(authFunction()).toBe('expected output');
});