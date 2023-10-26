// it will receive a json object with a map of entries, where the key is the rating, and the value is the counts of that rating
exports.handler = async (event) => {
    let body = JSON.parse(event.body)
    
    let ratings = body.ratings;
    let avg = 0;

    let total = 0;
    let totalCount = 0;

    for (let ratingValue in ratings) {
        totalCount += parseInt(ratings[ratingValue]);
        total += parseInt(ratingValue) * ratings[ratingValue];
    }

    avg = total / totalCount;

    const response = {
        statusCode: 200,
        body: JSON.stringify({
            'avg': avg,
            'totalCount': totalCount,
        }),
    };

    return response;
};
