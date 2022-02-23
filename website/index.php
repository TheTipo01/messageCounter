<!DOCTYPE html>
<html lang="it">
<head><title>Ghostpingers</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="css/bootstrap.min.css">
</head>
<body>
<div class="container"><h2>People who ping</h2>
    <p>List of peoples who ping other people:</p>
    <table class="table table-hover">
        <thead>
        <tr>
            <th>Mentioner</th>
            <th>Mentioned</th>
            <th>Date and time (in UTC)</th>
            <th>Guild</th>
            <th>Channel</th>
        </tr>
        </thead>
        <tbody>
        <?php

        $connection = new mysqli("ip", "user", "pass", "database");
        $query = "SELECT channels.name AS canale, servers.name AS serverName, pings.TIMESTAMP AS timestamp, users1.nickname AS menzionato, users2.nickname AS menzionatore FROM pings JOIN servers ON pings.serverId = servers.id JOIN users AS users1 ON pings.menzionatoId = users1.id JOIN users AS users2 ON pings.menzionatoreId = users2.id JOIN channels ON pings.channelId = channels.id ORDER BY pings.timestamp DESC";
        $result = $connection->query($query);

        if ($result->num_rows > 0) {
            while ($row = $result->fetch_assoc()) {
                echo "<tr><td>".$row["menzionatore"]."</td>"."<td>".$row["menzionato"]."</td>"."<td>".$row["timestamp"]."</td>"
                    ."<td>".$row["serverName"]."</td>"."<td>".$row["canale"]."</td>"."</tr>";
            }
        }
        $connection->close();
        ?>
        </tbody>
    </table>
</div>
</body>
</html>