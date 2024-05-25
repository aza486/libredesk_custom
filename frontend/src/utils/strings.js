// Adds titleCase property to string.
String.prototype.titleCase = function () {
    return this.toLowerCase().split(' ').map(function (word) {
        return word.charAt(0).toUpperCase() + word.slice(1);
    }).join(' ');
};