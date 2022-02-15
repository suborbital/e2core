var express = require('express');
var router = express.Router();

router.post('/', function(req, res, next) {
  let message = "Hello, " + req.body;

  console.info(message)

  res.send(message)
});

module.exports = router;
