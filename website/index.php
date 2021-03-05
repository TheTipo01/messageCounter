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

        $connection = mysqli_connect("ip", "user", "pass", "database");
        $query = "select `channels`.`name` AS `canale`,`server`.`name` AS `serverName`,`users`.`nickname` AS `menzionato`,`users2`.`nickname` AS `menzionatore`,`pings`.`timestamp` AS `TIMESTAMP` from ((((`channels` join `pings`) join `server`) join `users`) join `users` `users2`) where `pings`.`menzionatoreId` = `users2`.`id` and `pings`.`menzionatoId` = `users`.`id` and `pings`.`channelId` = `channels`.`id` and `pings`.`serverId` = `server`.`id` order by `pings`.`timestamp` desc";
        $result = mysqli_query($connection, $query);

        if (mysqli_num_rows($result) != 0) {
            while ($row = mysqli_fetch_array($result)) {
                echo "<tr>";
                echo "<td>$row[menzionatore]</td>";
                echo "<td>$row[menzionato]</td>";
                echo "<td>$row[TIMESTAMP]</td>";
                echo "<td>$row[serverName]</td>";
                echo "<td>$row[canale]</td>";
                echo "</tr>";
            }
        } else {
            mysqli_close($connection);
        }
        ?>
        </tbody>
    </table>
</div>
</body>
</html>